#!/bin/bash

start_iperf_server() {
	ns="$1"
	ip="$2"
	ip netns exec "$ns" iperf3 -s -B "$ip" -p 7575 \
		-D --logfile /tmp/iperf3.txt --forceflush
	ip netns exec "$ns" socat FILE:/dev/null TCP4-CONNECT:"$ip":7575,retry=10
	echo "iperf server is running"
}

stop_iperf() {
	ns="$1"
        ip netns exec "$ns" killall -9 iperf3 1>&2 2>/dev/null
}

start_iperf_client() {
	ns="$1"
	ip="$2"
	duration="$3"
	if ! [ "$duration" = 0 ]; then
		ip netns exec "$ns" iperf3 -c "$ip" -t "$duration" -p 7575
	fi
}

check_threshold() {
	field=$1
	value1=$2
	value2=$3
	threshold=$4
	retvalue=0

	if ! [ "$val1" = 0 ] && ! [ "$val2" = 0 ]; then
		diff=$(awk -v v1="$val1" -v v2="$val2" \
			'BEGIN{d=(100*(v2-v1)/v1); \
			if (d<0) d=d*(-1);printf("%2.2f\n", d)}')
		if awk "BEGIN {exit !($diff >= $threshold)}"; then
			echo "ERROR: $field $value1 $value2 $diff"
			retvalue=1
		fi
	else
		echo "ERROR: $field $value1 $value2"
		retvalue=1
	fi

	return "$retvalue"
}

compare() {
	echo "Checking that openstack-network-exporter statistics are ok"
	file1=$1
	file2=$2
	threshold=$3

	# Remove statistics that will not be checked
	set --
	# shellcheck disable=SC2013
	for skip in $(sed -En 's/([^,]+), *\<skip_field\>.*/\1/p' "$STATS_CONF"); do
		set -- "$@" -e "$skip"
	done
	echo "Filter: $*"
	grep -vw "$@" "$file1" > "$file1.filtered"
	grep -vw "$@" "$file2" > "$file2.filtered"

        # Check that same fields have been generated
	sed -En 's/^([a-z0-9_]+).*/\1/p' "$file1.filtered" > "$file1.fields"
	sed -En 's/^([a-z0-9_]+).*/\1/p' "$file2.filtered" > "$file2.fields"
	if ! diff -u "$file1.fields" "$file2.fields" >/dev/null; then
		echo "ERROR: Statistics set is not completed, Files have different fields"
		return 1
	fi

	# Check that values are similar (under a defined threshold)
	retvalue=0

	while read -r -u 4 line1 && read -r -u 5 line2; do
		if [ "$line1" = "$line2" ]; then
			continue
		fi
		field1="${line1% *}"
		field2="${line2% *}"
		val1="${line1#* }"
		val2="${line2#* }"
		if ! [ "$field1" = "$field2" ]; then
			echo "ERROR: Unexpected error, fields should coincide $field1 $field2"
			retvalue=1
			break
		fi
		field_base=$(echo "$field1" | sed -En 's/^([a-z0-9_]+).*/\1/p')
		stat_thr=$(sed -En "s/^$field_base, .*, ([0-9]+),.*/\\1/p" "$STATS_CONF")
		if [ -n "$stat_thr" ]; then
			echo "Set threshold $stat_thr for $field1"
		else
			stat_thr="$threshold"
		fi
		if ! check_threshold "$field1" "$val1" "$val2" "$stat_thr"; then
			retvalue=1
		fi
	done 4<"$file1.filtered" 5<"$file2.filtered"

	return "$retvalue"
}

get_stats() {
	file1="$1"
	file2="$2"
	options="$3"
	echo "Getting stats"
	curl -o "$file1" http://localhost:1981/metrics 2>/dev/null
	# shellcheck disable=SC2086
	"${TEST_DIR}"/get_ovs_stats.sh $options >"$file2"
	if ! [ -f "$file1" ] || ! [ -f "$file2" ]; then
		echo "Failed to get statistics"
		ls -ls "$file1" "$file2"
		return 1
	fi
	return 0
}

restart_openstack_network_exporter() {
	killall -9 openstack-network-exporter 2>/dev/null || true
	"$BASE_DIR"/openstack-network-exporter &
	sleep 5
}

get_environment() {
	namespaces=$(ip netns ls | awk '{print $1}')
	for ns in $namespaces; do
		ip=$(ip netns exec "$ns" ip a | grep -v "127.0.0.1" |
			sed -En 's/.*inet ([0-9.]+).*/\1/p')
		echo "$ns $ip"
	done
}

init_test() {
	TESTNAME=$(basename "$0")
	TESTNAME=${TESTNAME%.*}
	TESTDIR="$LOGS_DIR/$TESTNAME"

	mkdir -p "$TESTDIR"
}

end_test() {
	if [ "$1" != 0 ]; then
		echo "$TESTNAME: Testcase failed" |
			tee -a "$LOGS_DIR"/testcases.log
	else
		echo "$TESTNAME: Testcase passed" |
			tee -a "$LOGS_DIR"/testcases.log
	fi
	exit "$1"
}

ONE_CONFIG="/etc/openstack-network-exporter.yaml"
LOGS_DIR="$(dirname "$0")/logs"
TEST_DIR="$(dirname "$0")"
BASE_DIR="$TEST_DIR/../"
STATS_CONF="$TEST_DIR"/stats_conf.csv
THRESHOLD=2
IPERF_DURATION=10

ips=$(get_environment | tr '\n' ' ')
echo "ips      : $ips"

NS_0=$(echo "$ips" | awk '{print $1}')
IP_0=$(echo "$ips" | awk '{print $2}')
NS_1=$(echo "$ips" | awk '{print $3}')
IP_1=$(echo "$ips" | awk '{print $4}')
