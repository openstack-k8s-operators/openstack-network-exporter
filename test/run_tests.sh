#!/bin/bash
start_iperf_server()
{
   ns="${1}"
   ip="${2}"
   sudo ip netns exec "${ns}" iperf3 -s -B "${ip}" -p 7575 -D --logfile /tmp/iperf3.txt --forceflush
   while ! grep listening /tmp/iperf3.txt >/dev/null;do
      sleep 1
   done
   echo "iperf server is running"
}

start_iperf_client()
{
   ns="${1}"
   ip="${2}"
   iperf_duration="${3}"
   if [[ "${iperf_duration}" != "0" ]];then
      sudo ip netns exec "${ns}" iperf3 -c "${ip}" -t 40 -p 7575
   fi
}

filter_file()
{
   file="${1}"
   out="${2}"
   skipfiles="${3}"

   cmd="cat ${file}"
   for skip in ${skipfiles};do
      cmd="${cmd} | grep -v ${skip}"
   done
   echo "${cmd}" | bash > "${out}"
}

compare()
{
   echo "Checking that dataplane-node-exporter statistics are ok"
   file1=$1
   file2=$2
   threshold=$3

   skipstats_conf=$(dirname "$0")/stats_conf.csv
   skipstats=$(grep "skip_field" "${skipstats_conf}" | awk -F ',' '{print $1}' | tr '\n' ' ')
   echo "Filter: $skipstats"

   filter_file "${file1}" "${file1}".tmp1 "${skipstats}"
   filter_file "${file2}" "${file2}".tmp1 "${skipstats}"

   len1=$(wc -l "${file1}".tmp1 | awk '{print $1}')
   len2=$(wc -l "${file2}".tmp1 | awk '{print $1}')

   if [[ "${len1}" != "${len2}" ]];then
      echo "ERROR: Wrong number of statistics, files have different length ${len1} ${len2}"
      diff "${file1}".tmp1 "${file2}".tmp1
      return 1
   fi

   awk '{print $1}' "${file1}".tmp1 > "${file1}".tmp2
   awk '{print $1}' "${file2}".tmp1 > "${file2}".tmp2

   if ! diff "${file1}".tmp2 "${file2}".tmp2;then
     echo "ERROR: Statistics set is not completed, Files have different fields"
     diff "${file1}".tmp2 "${file2}".tmp2
     return 1
   fi

   retvalue=0
   while IFS= read -r -u 4 line1 && IFS= read -r -u 5 line2; do
      if [[ "${line1}" != "${line2}" ]];then
	 field1=$(echo "${line1}" | awk '{print $1}' | sed 's/ //g')
	 field2=$(echo "${line2}" | awk '{print $1}' | sed 's/ //g')
	 value1=$(echo "${line1}" | awk '{print $2}' | sed 's/ //g')
	 value2=$(echo "${line2}" | awk '{print $2}' | sed 's/ //g')
	 if [[ "${field1}" != "${field2}" ]];then
	    echo "ERROR: Unextected error, fields should coincide ${field1} ${field2}"
	    retvalue=1
	    break
	 fi 
	 field_base=$(echo "${field1}" | awk -F '{' '{print $1}')
         stat_threshold=$(grep "${field_base}" "${skipstats_conf}" | grep "set_threshold" | awk -F ',' '{print $3}')
	 if [[ "${stat_threshold}" != "" ]];then
	    echo "Setting threshold ${stat_threshold} for ${field1}"
	 else
            stat_threshold="${threshold}"
	 fi
	 if [[ "${value1}" != "0" && "${value2}" != "0" ]];then
	    diff=$(awk -v value1="${value1}" -v value2="${value2}" 'BEGIN{d=(100*(value2-value1)/value1);if (d<0) d=d*(-1);printf("%2.2f\n", d)}')
	    if awk "BEGIN {exit !($diff >= $stat_threshold)}"; then
	       if [[ ${retvalue} == 0 ]];then
		  echo "ERROR: Obtaing wrong values for some statistics"
		  retvalue=1
	       fi
	       echo "${field1} ${value1} ${value2} ${diff}"
	    fi
	 else
	   if [[ ${retvalue} == 0 ]];then
	      echo "ERROR: Obtaing wrong values for some statistics"
	      retvalue=1
	   fi
	   echo "${field1} ${value1} ${value2}"
	 fi
      fi
   done 4<"${file1}".tmp1 5<"${file2}".tmp1
   return "${retvalue}"
}

get_stats()
{
  file1="${1}"
  file2="${2}"
  options="${3}"
  echo "Getting stats"
  curl -o "${file1}" http://localhost:1981/metrics 2>/dev/null
  # shellcheck disable=SC2086
  "$(dirname "$0")"/get_ovs_stats.sh ${options} >"${file2}"
  if [[ ! -f "$file1" || ! -f "$file2" ]];then
     echo "Failed to get statistics"
     ls -ls "$file1" "$file2"
     return 1
  fi
  return 0
}

check_sudo()
{
   if sudo -n true 2>/dev/null; then
     return 0
   fi
   echo "sudo needed to run this script"
   return 1
}

restart_dataplane_node_exporter()
{
  sudo killall -9 dataplane-node-exporter
  sudo ./dataplane-node-exporter &
  sleep 5
}

test()
{
  ns="${1}"
  ip="${2}"
  dir="${3}"
  threshold="${4}"
  testname="${5}"
  iperf_duration="${6}"
  options="${7}"

  restart_dataplane_node_exporter
  file="${dir}/${testname}"
  start_iperf_client "${ns}" "${ip}" "${iperf_duration}"
  get_stats "${file}_1" "${file}_2" "${options}"
  compare "${file}_1" "${file}_2" "${threshold}"
  return $?
}

test1()
{
  echo "Test1: Get statistics with default configuration"
  sudo rm /etc/dataplane-node-exporter.yaml 2>/dev/null
  test "$@" "test1" "10"
  return $?
}

test2()
{
  echo "Test2: Get statistics with only with some collectors"
  echo "collectors: [interface, memory]" | sudo tee /etc/dataplane-node-exporter.yaml
  test "$@" "test2" "10" "-c interface:memory"
  return $?
}

test3()
{
  echo "Test3: Get statistics with only with some collectors and metricsets"
  echo "collectors: [interface, memory]" | sudo tee /etc/dataplane-node-exporter.yaml
  echo "metric-sets: [errors, counters]" | sudo tee -a /etc/dataplane-node-exporter.yaml
  test "$@" "test3" "10" "-c interface:memory -m errors:counters"
  return $?
}

run_tests()
{
   ret=0
   testcases=$(echo "${1}" | tr ':' ' ')
   ns="${2}"
   ip="${3}"
   threshold="${4}"
   dir="logs"
   mkdir "${dir}" 2>/dev/null
   for test in ${testcases};do
      $test "${ns}" "${ip}" "${dir}" "${threshold}"
      ret_test=$?
      if [[ "${ret_test}" != 0 ]];then
         echo "${test}: Testcase failed"
	 ret="${ret_test}"
      else
         echo "${test}: Testcase passed"
      fi
   done
   return "${ret}"
}

get_environment()
{
   namespaces=$(sudo ip netns ls | awk '{print $1}')
   for ns in ${namespaces};do
      ip=$(sudo ip netns exec "${ns}" ip a | grep "inet " | awk '{print $2}' | awk -F '/' '{print $1}')
      echo "${ns} ${ip}"
   done
}

help()
{
   echo "Run testcases"
   echo "run_tests.sh [-h | -t testcases ]"
   echo "   -h for this help"
   echo "   -t for testcases list separated by :"
   echo "   -r threshold numeric values in %, default 2"
   echo "Sudo needed"
}

while getopts h?t:r: flag
do
    case "${flag}" in
        t) testcases=${OPTARG};;
	r) threshold=${OPTARG};;
        h|\?) help; exit 0;;
        *) help; exit 0;;
    esac
done
testcases=${testcases:-test1:test2:test3}
threshold=${threshold:-2}

echo "testcases: ${testcases}"
echo "threshold: ${threshold}"

ips=$(get_environment | tr '\n' ' ')
echo "ips      : ${ips}"

ns0=$(echo "${ips}" | awk '{print $1}')
ip0=$(echo "${ips}" | awk '{print $2}')
ns1=$(echo "${ips}" | awk '{print $3}')

if ! check_sudo;then
   exit 1
fi

start_iperf_server "${ns0}" "${ip0}"

run_tests "${testcases}" "${ns1}" "${ip0}" "${threshold}"
ret_test=$?

killall -9 iperf3 1>&2 2>/dev/null
exit "${ret_test}"
