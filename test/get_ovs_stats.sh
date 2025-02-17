#!/bin/bash
get_bridge_stats()
{
   bridges=$(sudo ovs-vsctl list-br)
   for bridge in ${bridges};do
      datapath_type=$(sudo ovs-vsctl get bridge "${bridge}" datapath_type)
      flows=$(sudo ovs-ofctl dump-aggregate "${bridge}" | tr ' ' '\n' | grep flow_count | awk -F '=' '{print $2}')
      ports=$(sudo ovs-ofctl show "${bridge}" | grep -c addr)
      echo "ovs_bridge_flow_count{bridge=\"${bridge}\",datapath_type=\"${datapath_type}\"} ${flows}"
      echo "ovs_bridge_port_count{bridge=\"${bridge}\",datapath_type=\"${datapath_type}\"} ${ports}"
   done
}

get_vswitch_stats()
{
   ovs_version=$(sudo ovs-vsctl get Open_vSwitch . ovs_version)
   dpdk_version=$(sudo ovs-vsctl get Open_vSwitch . dpdk_version)
   db_version=$(sudo ovs-vsctl get Open_vSwitch . db_version)
   dpdk_initialized=$(sudo ovs-vsctl get Open_vSwitch . dpdk_initialized)
   [[ ${dpdk_initialized} == true ]] && dpdk_initialized=1 || dpdk_initialized=0

   echo "ovs_build_info{db_version=${db_version},dpdk_version=${dpdk_version},ovs_version=${ovs_version}} 1"
   echo "ovs_dpdk_initialized ${dpdk_initialized}"
}

get_coverage_stats()
{
   sudo ovs-appctl  coverage/show  | grep total | awk '{print "ovs_coverage_"$1"_total",$6}'
}

get_datapath_stats()
{
   sudo ovs-appctl  dpctl/show | grep -v port | while IFS= read -r line; do
     if echo "${line}" | grep '@' >/dev/null;then
       datapath_type=$(echo "${line}" | awk -F "@" '{print $1}')
       name=$(echo "${line}" | awk -F "@" '{print $2}' | tr -d ":")
     elif echo "${line}" | grep 'lookups' >/dev/null;then
       hit=$(echo "${line}" | tr ' ' '\n' | grep hit | awk -F ':' '{print $2}')
       missed=$(echo "${line}" | tr ' ' '\n' | grep missed | awk -F ':' '{print $2}')
       lost=$(echo "${line}" | tr ' ' '\n' | grep lost | awk -F ':' '{print $2}')
       echo "ovs_datapath_lookup_hits_total{name=\"${name}\",type=\"${datapath_type}\"} ${hit}"
       echo "ovs_datapath_lookup_missed_total{name=\"${name}\",type=\"${datapath_type}\"} ${missed}"
       echo "ovs_datapath_lookup_lost_total{name=\"${name}\",type=\"${datapath_type}\"} ${lost}"
     elif echo "${line}" | grep 'flows' >/dev/null;then
       flows=$(echo "${line}" | awk -F ':' '{print $2}')
       echo "ovs_datapath_flows_total{name=\"${name}\",type=\"${datapath_type}\"} ${flows}"
     fi
   done
}

get_interface_stats()
{
   fields="admin_state link_resets link_speed:link_speed_bps link_state mtu:mtu_bytes"
   bridges=$(sudo ovs-vsctl list-br)
   for bridge in ${bridges};do
     ports=$(sudo ovs-ofctl show "${bridge}" | grep addr | awk -F "(" '{print $2}' | awk -F ")" '{print $1}')
     for port in ${ports};do
        type=$(sudo ovs-vsctl get interface "${port}" type)
       [[ "${type}" == '""' ]] && type=''
        for field in ${fields};do
           field1=$(echo "${field}" | awk -F ':' '{print $1}')
           field2=$(echo "${field}" | awk -F ':' '{print $2}')
           [[ "${field2}" == "" ]] && field2=${field1}
           val=$(sudo ovs-vsctl get interface "${port}" "${field1}")
	   if [[ "${val}" == "down" ]];then val="0";
           elif [[ "${val}" == "up" ]];then val="1";
           fi
           echo "ovs_interface_${field2}{bridge=\"${bridge}\",interface=\"${port}\",port=\"${port}\",type=\"${type}\"} ${val}"
        done
	rx_dropped="0"
	tx_errors_print="1"
        sudo ovs-vsctl get interface "${port}" statistics | tr ',' '\n' | tr -d '{}' | sort | while IFS= read -r line; do
           stat=$(echo "${line}" | awk -F '=' '{print $1}' | sed 's/ //g')
           val=$(echo "${line}" | awk -F '=' '{print $2}' | sed 's/ //g')
	   queue=""
	   if grep -E -o "rx_q[[:digit:]+]_guest_notifications" "${stat}" 2>/dev/null;then
              queue=$(grep -E -o "rx_q[[:digit:]+]_guest_notifications" "${stat}" | sed 's/rx_q//g' | sed 's/_guest_notifications//g')
	      stat="ovs_interface_rx_guest_notifications"
           elif grep -E -o "tx_q[[:digit:]+]_guest_notifications" "${stat}" 2>/dev/null;then
              queue=$(grep -E -o "tx_q[[:digit:]+]_guest_notifications" "${stat}" | sed 's/tx_q//g' | sed 's/_guest_notifications//g')
	      stat="ovs_interface_tx_guest_notifications"
           elif grep -E -o "rx_q[[:digit:]+]_good_packets" "${stat}" 2>/dev/null;then
              queue=$(grep -E -o "rx_q[[:digit:]+]_good_packets" "${stat}" | sed 's/rx_q//g' | sed 's/_good_packets//g')
	      stat="ovs_interface_rx_good_packets"
           elif grep -E -o "tx_q[[:digit:]+]_good_packets" "${stat}" 2>/dev/null;then
              queue=$(grep -E -o "tx_q[[:digit:]+]_good_packets" "${stat}" | sed 's/tx_q//g' | sed 's/_good_packets//g')
	      stat="ovs_interface_tx_good_packets"
           elif grep -E -o "rx_q[[:digit:]+]_multicast_packets" "${stat}" 2>/dev/null;then
              queue=$(grep -E -o "rx_q[[:digit:]+]_multicast_packets" "${stat}" | sed 's/rx_q//g' | sed 's/_multicast_packets//g')
	      stat="ovs_interface_rx_multicast_packets"
           elif grep -E -o "tx_q[[:digit:]+]_multicast_packets" "${stat}" 2>/dev/null;then
              queue=$(grep -E -o "tx_q[[:digit:]+]_multicast_packets" "${stat}" | sed 's/tx_q//g' | sed 's/_multicast_packets//g')
	      stat="ovs_interface_tx_multicast_packets"
           elif [[ "${stat}" == "ovs_tx_retries" ]];then
	      stat="ovs_interface_tx_retries"
           elif [[ "${stat}" == "rx_dropped" ]];then
	      rx_dropped="${val}"
	      continue
           elif [[ "${stat}" == "rx_missed_errors" ]];then
	      stat="ovs_interface_rx_dropped"
	      if [ "${val}" -le 0 ];then
	          val="${rx_dropped}"
	      fi
           elif [[ "${stat}" == "ovs_tx_failure_drops" ]];then
	      if [ "${val}" -gt 0 ];then
	         stat="ovs_interface_tx_errors"
		 tx_errors_print="0"
   	      else
		 continue
	      fi
           elif [[ "${stat}" == "tx_errors" ]];then
	      stat="ovs_interface_tx_errors"
	      if [[ "${tx_errors_print}" == "0" ]];then
	          continue
	      fi
           elif echo "rx_packets rx_bytes rx_errors tx_packets tx_bytes" | tr ' ' '\n' | grep "${stat}" >/dev/null;then
              stat="ovs_interface_${stat}"
           else
             continue
           fi
	   if [[ "${queue}" != "" ]];then
              echo "${stat}{bridge=\"${bridge}\",interface=\"${port}\",port=\"${port}\",type=\"${type}\",queue=\"${queue}\"} ${val}"
	   else
              echo "${stat}{bridge=\"${bridge}\",interface=\"${port}\",port=\"${port}\",type=\"${type}\"} ${val}"
           fi
        done
     done
   done
}

get_memory_stats()
{
   memory_stats=$(sudo ovs-appctl memory/show)
   for stat in ${memory_stats};do
      stat_name=$(echo "${stat}" | awk -F ':' '{print $1}')
      stat_val=$(echo "${stat}" | awk -F ':' '{print $2}')
      echo "ovs_memory_${stat_name}_total ${stat_val}"
   done
}

get_pmd_rxq_stats()
{
   sudo ovs-appctl dpif-netdev/pmd-rxq-show | while IFS= read -r line; do
      if echo "${line}" | grep '^pmd' >/dev/null;then
         numa_id=$(echo "${line}" | tr ' ' '\n' | grep -A 1 numa_id | tail -1)
         core_id=$(echo "${line}" | tr ' ' '\n' | sed 's/://g' | grep -A 1 core_id | tail -1)
      elif echo "${line}" | grep 'port' > /dev/null;then
         rxq=$(echo "${line}" | sed 's/:[ ][ ]*/:/g' | tr ' ' '\n' | grep queue-id | awk -F ':' '{print $2}')
         interface=$(echo "${line}" | sed 's/:[ ][ ]*/:/g' | tr ' ' '\n' | grep port | awk -F ':' '{print $2}')
         usage=$(echo "${line}" | sed 's/:[ ][ ]*/:/g' | tr ' ' '\n' | grep usage | awk -F ':' '{print $2}')
         enabled=$(echo "${line}" | sed 's/:[ ][ ]*/:/g' | tr ' ' '\n' | grep enabled)
         [[ "${enabled}" != "" ]] && enabled=1 || enabled=0
         echo "ovs_pmd_rxq_enabled{cpu=\"${core_id}\",interface=\"${interface}\",numa=\"${numa_id}\",rxq=\"${rxq}\"} ${enabled}"
         echo "ovs_pmd_rxq_usage{cpu=\"${core_id}\",interface=\"${interface}\",numa=\"${numa_id}\",rxq=\"${rxq}\"} ${usage}"
     elif echo "${line}" | grep 'overhead' > /dev/null;then
         overhead=$(echo "${line}" | awk -F ':' '{print $2}' | awk '{print $1}')
         echo "ovs_pmd_cpu_overhead{cpu=\"${core_id}\",numa=\"${numa_id}\"} ${overhead}"
     elif echo "${line}" | grep 'isolated' > /dev/null;then
         isolated=$(echo "${line}" | awk -F ':' '{print $2}')
         [[ ${isolated} == "true" ]] && isolated=1 || isolated=0
         echo "ovs_pmd_cpu_isolated{cpu=\"${core_id}\",numa=\"${numa_id}\"} ${isolated}"
     fi
   done
}

get_pmd_perf_stats()
{
   core_list_map=""
   while IFS= read -r line; do
      if echo "${line}" | grep '^pmd' >/dev/null;then
         numa_id=$(echo "${line}" | tr ' ' '\n' | grep -A 1 numa_id | tail -1)
         core_id=$(echo "${line}" | tr ' ' '\n' | sed 's/://g' | grep -A 1 core_id | tail -1)
	 core_list_map="${core_list_map} ${core_id}:${numa_id}"
     else
        stat=""
        if echo "${line}" | tr -d ' ' | grep '^Iterations' > /dev/null;then stat="ovs_pmd_total_iterations";
        elif echo "${line}" | grep 'Used TSC cycles' > /dev/null;then stat="ovs_pmd_used_tsc_cycles";
        elif echo "${line}" | grep 'idle iterations' > /dev/null;then stat="ovs_pmd_idle_iterations";
        elif echo "${line}" | grep 'busy iterations' > /dev/null;then stat="ovs_pmd_busy_iterations";
        elif echo "${line}" | grep 'sleep iterations' > /dev/null;then stat="ovs_pmd_sleep_iterations";
        elif echo "${line}" | grep 'Sleep time (us)' > /dev/null;then stat="ovs_pmd_sleep_microseconds";
        elif echo "${line}" | grep 'Rx packets' > /dev/null;then stat="ovs_pmd_rx_packets";
        elif echo "${line}" | grep 'Datapath passes' > /dev/null;then stat="ovs_pmd_datapath_passes";
        elif echo "${line}" | grep 'PHWOL hits' > /dev/null;then stat="ovs_pmd_phwol_hits";
        elif echo "${line}" | grep 'MFEX Opt hits' > /dev/null;then stat="ovs_pmd_mfex_opt_hits";
        elif echo "${line}" | grep 'Simple Match hits' > /dev/null;then stat="ovs_pmd_simple_match_hits";
        elif echo "${line}" | grep 'EMC hits' > /dev/null;then stat="ovs_pmd_emc_hits";
        elif echo "${line}" | grep 'SMC hits' > /dev/null;then stat="ovs_pmd_smc_hits";
        elif echo "${line}" | grep 'Megaflow hits' > /dev/null;then stat="ovs_pmd_megaflow_hits";
        elif echo "${line}" | grep 'Upcalls' > /dev/null;then stat="ovs_pmd_total_upcalls";
        elif echo "${line}" | grep 'Lost upcalls' > /dev/null;then stat="ovs_pmd_lost_upcalls";
        elif echo "${line}" | grep 'Tx packets' > /dev/null;then stat="ovs_pmd_tx_packets";
        elif echo "${line}" | grep 'Tx batches' > /dev/null;then stat="ovs_pmd_tx_batches";
        fi
	value=$(echo "${line}" | awk -F ':' '{print $2}' | awk -F '(' '{print $1}' | tr -d ' ')
        if [[ "${stat}" != "" && "${value}" != "" ]];then
           echo "${stat}{cpu=\"${core_id}\",numa=\"${numa_id}\"} ${value}"
        fi
      fi
   done < <(sudo ovs-appctl dpif-netdev/pmd-perf-show)
   ovs_pid=$(cat /run/openvswitch/ovs-vswitchd.pid)
   pmd=""
   sudo find /proc/"${ovs_pid}"/task -name status -exec egrep  "Name|voluntary_ctxt_switches|nonVolCtxSwitches" {} \; | while IFS= read -r line; do
      token1=$(echo "${line}" | awk -F ':' '{print $1}')
      token2=$(echo "${line}" | awk -F ':' '{print $2}')
      if [[ "${token1}" == "Name" ]];then
         pmd=""
	 if echo "${token2}" | grep "pmd" > /dev/null;then
            pmd=$(echo "${token2}" | sed 's/pmd-c//g' | sed 's/\// /g' | awk '{print $1}' | xargs printf "%d\n")
         fi
      elif [[ "${pmd}" != "" ]];then
         if [[ "${token1}" == "voluntary_ctxt_switches" ]];then
            stat="ovs_pmd_context_switches"
         elif [[ "${token1}" == "nonvoluntary_ctxt_switches" ]];then
            stat="ovs_pmd_nonvol_context_switches"
	 fi
	 value="${token2}"
	 numa_id=$(echo "${core_list_map}" | tr ' ' '\n' | grep "^${pmd}:" | awk -F ':' '{print $2}')
         echo "${stat}{cpu=\"${pmd}\",numa=\"${numa_id}\"} ${value}"
      fi
   done
}

get_stats()
{
   collectors=$(echo "$1" | tr ':' ' ')
   for collector in ${collectors};do
      [[ "${collector}" == "bridge" ]] && get_bridge_stats
      [[ "${collector}" == "vswitch" ]] && get_vswitch_stats
      [[ "${collector}" == "coverage" ]] && get_coverage_stats
      [[ "${collector}" == "datapath" ]] && get_datapath_stats
      [[ "${collector}" == "interface" ]] && get_interface_stats
      [[ "${collector}" == "memory" ]] && get_memory_stats
      [[ "${collector}" == "pmd-rxq" ]] && get_pmd_rxq_stats
      [[ "${collector}" == "pmd-perf" ]] && get_pmd_perf_stats
   done
}

filter()
{
   statscsv=$1
   metricsets=$2
   last=""
   while IFS= read -r line; do
      metric=$(echo "${line}" | awk '{print $1}' | awk -F '{' '{print $1}')
      if grep "${metric}" "${statscsv}" >/dev/null;then
         metric_type=$(grep "${metric}" "${statscsv}" | awk -F ';' '{print $3}')
         data_type=$(grep "${metric}" "${statscsv}" | awk -F ';' '{print $4}')
         desc=$(grep "${metric}" "${statscsv}" | awk -F ';' '{print $6}')
         if echo "${metricsets}" | tr ':' '\n' | grep "${metric_type}" >/dev/null;then
            if [[ "${last}" != "${metric}" ]];then
	       last=${metric}
	       echo "# HELP ${metric} ${desc}"
	       echo "# TYPE ${metric} ${data_type}"
            fi
	    f2=$(echo "${line}" | awk '{print $NF}')
	   [[ "${f2}" -ge "1000000" ]] && f2=$(printf "%e\n" "${f2}")
	   echo "${line}" | awk -v val="${f2}" '{$NF=val; print}'
         fi
      fi
   done
}

help()
{
   echo "Get ovs stats"
   echo "get_ovs_stats.sh [-h | -c collectors -m metricsets]"
   echo "   -h for this help"
   echo "   -c for collector list separated by :"
   echo "   -m for metric sets separated by :"
   echo "Supported collectors are: bridge coverage datapath interface memory pmd-perf pmd-rxq vswitch"
   echo "Supported metric sets are: base errors perf counters debug"
   echo "Sudo needed"
} 

check_sudo()
{
   if sudo -n true 2>/dev/null; then
     return 0
   fi
   echo "sudo needed to run this script"
   return 1
}

while getopts h?c:m: flag
do
    case "${flag}" in
        c) collectors=${OPTARG};;
        m) metricsets=${OPTARG};;
	h|\?) help; exit 0;;
	*) help; exit 0;;
    esac
done

if ! check_sudo;then
   exit 1
fi
statscsv=$(dirname "$0")/stats.csv
collectors=${collectors:-bridge:coverage:datapath:interface:memory:pmd-perf:pmd-rxq:vswitch}
metricsets=${metricsets:-base:errors:perf:counters}
get_stats "${collectors}" | sort | filter "${statscsv}" "${metricsets}" 
exit 0
