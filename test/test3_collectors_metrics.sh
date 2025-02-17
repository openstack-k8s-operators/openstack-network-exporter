#!/bin/bash

ENV_TEST_PATH=$(dirname "$0")/env_test.sh
# shellcheck source=test/env_test.sh
source "$ENV_TEST_PATH"
init_test
options="-c interface:memory -m errors:counters"

echo "$TESTNAME: Get statistics with only with some collectors and metricsets"
echo "collectors: [interface, memory]" | tee "$ONE_CONFIG"
echo "metric-sets: [errors, counters]" | tee -a "$ONE_CONFIG"
restart_openstack_network_exporter
start_iperf_server "$NS_0" "$IP_0"
start_iperf_client "$NS_1" "$IP_0" "$IPERF_DURATION"
stop_iperf "$NS_0"
get_stats "$TESTDIR/op_net_ex" "$TESTDIR/test" "$options"
compare "$TESTDIR/op_net_ex" "$TESTDIR/test" "$THRESHOLD"
end_test $?
