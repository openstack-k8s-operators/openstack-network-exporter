#!/bin/bash

get_bridge_stats() {
	bridges=$(ovs-vsctl list-br)
	for bridge in $bridges; do
		dp_type=$(ovs-vsctl get bridge "$bridge" datapath_type)
		flows=$(ovs-ofctl dump-aggregate "$bridge" |
			sed -n 's/.*flow_count=//p')
		ports=$(ovs-ofctl show "$bridge" | grep -c addr)
		cat << EOF
ovs_bridge_flow_count{bridge="$bridge",datapath_type="$dp_type"} $flows
ovs_bridge_port_count{bridge="$bridge",datapath_type="$dp_type"} $ports
EOF
	done
}

get_vswitch_stats() {
	ovs_version=$(ovs-vsctl get o . ovs_version)
	dpdk_version=$(ovs-vsctl get o . dpdk_version)
	db_version=$(ovs-vsctl get o . db_version)
	dpdk_initialized=$(ovs-vsctl get o . dpdk_initialized)
	if [ "$dpdk_initialized" == true ]; then
		dpdk_initialized=1
	else
		dpdk_initialized=0
	fi

	cat << EOF
ovs_build_info{db_version=$db_version,dpdk_version=$dpdk_version,ovs_version=$ovs_version} 1
ovs_dpdk_initialized $dpdk_initialized
EOF
}

get_coverage_stats() {
	ovs-appctl  coverage/show | awk '/total/ { print "ovs_coverage_"$1"_total",$6 }'
}

get_datapath_stats() {
	ovs-appctl dpctl/show | grep -v port |
	while read -r line; do
		case "$line" in
		*@*)
                        datapath_type=${line%@*}
                        datapath_type=${datapath_type#:*}
                        name=${line#*@}
                        name=${name%:*}
			;;
		*lookups:*)
			hit=$(echo "$line" | sed -En 's/.*hit:([0-9]+).*/\1/p')
			miss=$(echo "$line" |
				sed -En 's/.*missed:([0-9]+).*/\1/p')
			lost=$(echo "$line" |
				sed -En 's/.*lost:([0-9]+).*/\1/p')
			cat <<EOF
ovs_datapath_lookup_hits_total{name="$name",type="$datapath_type"} $hit
ovs_datapath_lookup_missed_total{name="$name",type="$datapath_type"} $miss
ovs_datapath_lookup_lost_total{name="$name",type="$datapath_type"} $lost
EOF
			;;
		*flows:*)
			cat <<EOF
ovs_datapath_flows_total{name="$name",type="$datapath_type"} ${line#*flows: }
EOF
			;;
		esac
	done
}

parse_port_stats() {
	port="$1"
	declare -A port_stats=()
	for stat in $(ovs-vsctl get interface "$port" statistics | tr -d '{},'); do
		port_stats[${stat%=*}]=${stat#*=}
	done

	for stat in "${!port_stats[@]}"; do
		queue=""
		val="${port_stats[$stat]}"
		case "$stat" in
		rx_q*_guest_notifications)
			queue=$(echo "$stat" |
				sed -En 's/rx_q([0-9]+)_guest_notifications/\1/p')
			stat="ovs_interface_rx_guest_notifications"
			;;
		tx_q*_guest_notifications)
			queue=$(echo "$stat" |
				sed -En 's/tx_q([0-9]+)_guest_notifications/\1/p')
			stat="ovs_interface_tx_guest_notifications"
			;;
		rx_q*_good_packets)
			queue=$(echo "$stat" |
				sed -En 's/rx_q([0-9]+)_good_packets/\1/p')
			stat="ovs_interface_rx_good_packets"
			;;
		tx_q*_good_packets)
			queue=$(echo "$stat" |
				sed -En 's/tx_q([0-9]+)_good_packets/\1/p')
			stat="ovs_interface_tx_good_packets"
			;;
		rx_q*_multicast_packets)
			queue=$(echo "$stat" |
				sed -En 's/rx_q([0-9]+)_multicast_packets/\1/p')
			stat="ovs_interface_rx_multicast_packets"
			;;
		tx_q*_multicaset_packets)
			queue=$(echo "$stat" |
				sed -En 's/tx_q([0-9]+)_multicast_packets/\1/p')
			stat="ovs_interface_tx_multicast_packets"
			;;
		ovs_tx_retries)
			stat="ovs_interface_tx_retries"
			;;
		rx_dropped)
			stat="ovs_interface_rx_dropped"
			rx_miss=${port_stats["rx_missed_errors"]}
			if [ -n "$rx_miss" ] && [ "$rx_miss" -gt 0 ]; then
			   val="$rx_miss"
			fi
			;;
		tx_errors)
			stat="ovs_interface_tx_errors"
			tx_fdrops=${port_stats["ovs_tx_failure_drops"]}
			if [ -n "$tx_drops" ] && [ "$tx_drops" -gt 0 ]; then
			   val="$tx_drops"
			fi
			;;
		rx_packets|rx_bytes|rx_errors|tx_packets|tx_bytes)
			stat="ovs_interface_$stat"
			;;
		*)
			continue
			;;
		esac

		if [ "$queue" != "" ]; then
			cat << EOF
$stat{bridge="$bridge",interface="$port",port="$port",type="$type",queue="$queue"} $val
EOF
		else
			cat << EOF
$stat{bridge="$bridge",interface="$port",port="$port",type="$type"} $val
EOF
		fi
	done
}

get_interface_stats() {
	declare -A fields=( \
		["admin_state"]="admin_state"
		["link_resets"]="link_resets"
		["link_speed"]="link_speed_bps"
		["link_state"]="link_state"
		["mtu"]="mtu_bytes"
	)
	declare -A val_enc=( \
		["down"]=0
		["up"]=1
	)
	for bridge in $(ovs-vsctl list-br); do
		ports=$(ovs-ofctl show "${bridge}" | grep addr |
		sed -En 's/.*\((.*)\).*/\1/p')
		for port in $ports; do
			type=$(ovs-vsctl get interface "$port" type)
			if [ "$type" = '""' ]; then
				type=system
			fi
			for field in "${!fields[@]}"; do
				val=$(ovs-vsctl get interface \
					"$port" "$field")
				if [ -v "val_enc[$val]" ] ; then
					val="${val_enc[$val]}"
                                fi
				cat << EOF
ovs_interface_${fields[$field]}{bridge="$bridge",interface="$port",port="$port",type="$type"} $val
EOF
			done
			parse_port_stats "$port"
		done
	done
}

get_memory_stats() {
	for stat in $(ovs-appctl memory/show); do
		stat_name=${stat%:*}
		stat_val=${stat#*:}
		cat << EOF
ovs_memory_${stat_name}_total $stat_val
EOF
	done
}

get_pmd_rxq_stats() {
	ovs-appctl dpif-netdev/pmd-rxq-show |
	while read -r line; do
		case "$line" in
		pmd*)
			numa_id=$(echo "$line" | sed -En 's/.*numa_id ([0-9]+).*/\1/p')
			core_id=$(echo "$line" | sed -En 's/.*core_id ([0-9]+):/\1/p')
			;;
		*port*)
			rxq=$(echo "${line}" | sed -En 's/.*queue-id: *([0-9]+).*/\1/p')
			interface=$(echo "${line}" | sed -En 's/.*port: *([a-zA-Z0-9]+).*/\1/p')
			# Truncate float decimals, if any
			usage=$(echo "${line}" | sed -En 's/.*usage: *([0-9]+).*/\1/p')
			enab=$(echo "${line}" | sed -En 's/.*\(enabled\).*/1/p')
			cat << EOF
ovs_pmd_rxq_enabled{cpu="$core_id",interface="$interface",numa="$numa_id",rxq="$rxq"} ${enab:-0}
ovs_pmd_rxq_usage{cpu="$core_id",interface="$interface",numa="$numa_id",rxq="$rxq"} $usage
EOF
			;;
		*overhead*)
			overhead=$(echo "$line" | sed -En 's/.*overhead: *([0-9]+).*/\1/p')
			cat << EOF
ovs_pmd_cpu_overhead{cpu="$core_id",numa="$numa_id"} $overhead
EOF
			;;
		*isolated*)
			isolated=$(echo "${line}" | sed -En 's/.*isolated[: ]*true.*/1/p')
			cat << EOF
ovs_pmd_cpu_isolated{cpu="$core_id",numa="$numa_id"} ${isolated:-0}
EOF
			;;
		esac
	done
}

get_pmd_perf_stats() {
	declare -A core_list_map
	declare -A perf_stats=( \
		["Iterations"]="ovs_pmd_total_iterations"
		["- Used TSC cycles"]="ovs_pmd_used_tsc_cycles"
		["- idle iterations"]="ovs_pmd_idle_iterations"
		["- busy iterations"]="ovs_pmd_busy_iterations"
		["- sleep iterations"]="ovs_pmd_sleep_iterations"
		["Sleep time (us)"]="ovs_pmd_sleep_microseconds"
		["Rx packets"]="ovs_pmd_rx_packets"
		["Datapath passes"]="ovs_pmd_datapath_passes"
		["- PHWOL hits"]="ovs_pmd_phwol_hits"
		["- MFEX Opt hits"]="ovs_pmd_mfex_opt_hits"
		["- Simple Match hits"]="ovs_pmd_simple_match_hits"
		["- EMC hits"]="ovs_pmd_emc_hits"
		["- SMC hits"]="ovs_pmd_smc_hits"
		["- Megaflow hits"]="ovs_pmd_megaflow_hits"
		["- Upcalls"]="ovs_pmd_total_upcalls"
		["- Lost upcalls"]="ovs_pmd_lost_upcalls"
		["Tx packets"]="ovs_pmd_tx_packets"
		["Tx batches"]="ovs_pmd_tx_batches"
	)
	while read -r line; do
		if echo "$line" | grep -q '^pmd'; then
			numa_id=$(echo "$line" | sed -En 's/.*numa_id ([0-9]+).*/\1/p')
			core_id=$(echo "$line" | sed -En 's/.*core_id ([0-9]+).*/\1/p')
			core_list_map["$core_id"]="$numa_id"
		elif [ "$line" != "" ]; then
			name=$(echo "$line" | sed -En 's/^[[:space:]]*([^:]+):.*/\1/p')
			value=$(echo "$line" | sed -En 's/[^:]+:[[:space:]]*([0-9]+).*/\1/p')
			stat=${perf_stats[$name]}
			if [ -n "$stat" ] && [ -n "$value" ]; then
				cat << EOF
$stat{cpu="$core_id",numa="$numa_id"} $value
EOF
			fi
		fi
	done < <(ovs-appctl dpif-netdev/pmd-perf-show)

	ovs_pid=$(cat /run/openvswitch/ovs-vswitchd.pid)
        for status in /proc/"$ovs_pid"/task/*/status; do
		cpu=""
		numa=""
                while read -r line; do
                        case "$line" in
                        Name:*pmd*)
                                 cpu=$(echo "$line" |
                                 sed -En 's/.*pmd-c[0]*([1-9]+).*/\1/p')
                                 numa=${core_list_map[$cpu]}
                                 ;;
                        voluntary_ctxt_switches:*)
                                 if [ "$cpu" == "" ] || [ "$numa" == "" ]; then
                                         continue
                                 fi
                                 value="$((${line#*:}))"
                                 cat <<EOF
ovs_pmd_context_switches{cpu="$cpu",numa="$numa"} $value
EOF
                                 ;;
                        nonvoluntary_ctxt_switches:*)
                                 if [ "$cpu" == "" ] || [ "$numa" == "" ]; then
                                         continue
                                 fi
                                 value="$((${line#*:}))"
                                 cat <<EOF
ovs_pmd_nonvol_context_switches{cpu="$cpu",numa="$numa"} $value
EOF
                                 ;;
                        esac
                done < "$status"
        done
}

get_stats() {
	for collector in "$@"; do
		case "$collector" in
		bridge)
			get_bridge_stats
			;;
		vswitch)
			get_vswitch_stats
			;;
		coverage)
			get_coverage_stats
			;;
		datapath)
			get_datapath_stats
			;;
		interface)
			get_interface_stats
			;;
		memory)
			get_memory_stats
			;;
		pmd-rxq)
			get_pmd_rxq_stats
			;;
		pmd-perf)
			get_pmd_perf_stats
			;;
		*)
			echo "unsupported collector" >&2
			exit 1
			;;
		esac
	done
}

filter() {
	statscsv=$1
	metricsets=$2
	last=""
	while read -r line; do
		metric=$(echo "$line" | sed -En 's/^([a-zA-Z0-9_]+).*/\1/p')
		if grep -q "$metric" "$statscsv"; then
			metric_type=$(grep "$metric" "$statscsv" | awk -F ';' '{print $3}')
			data_type=$(grep "$metric" "$statscsv" | awk -F ';' '{print $4}')
			desc=$(grep "$metric" "$statscsv" | awk -F ';' '{print $6}')
			if echo "$metricsets" | tr ' ' '\n' | grep -q "$metric_type"; then
				if [ "$last" != "$metric" ]; then
					last=$metric
					echo "# HELP $metric $desc"
					echo "# TYPE $metric $data_type"
				fi
				value=${line##* }
				if [ "$value" -ge 1000000 ]; then
					# values greater than 1M are expected in scientific notation
					value=$(printf '%e' "$value")
				fi
				echo "${line% *} $value"
			fi
		fi
	done
}

help() {
	echo "Get ovs stats"
	echo "get_ovs_stats.sh [-h | -c collectors -m metricsets]"
	echo "   -h for this help"
	echo "   -c for collector list separated by :"
	echo "   -m for metric sets separated by :"
	echo "Supported collectors are: bridge coverage datapath interface"
	echo "                          memory pmd-perf pmd-rxq vswitch"
	echo "Supported metric sets are: base errors perf counters debug"
	echo "Sudo needed"
}

while getopts h?c:m: flag; do
	case "$flag" in
	c)
		collectors=${OPTARG//:/ }
		;;
	m)
		metricsets=${OPTARG//:/ }
		;;
	h|\?)
		help;
		exit 0
		;;
	*)
		help;
		exit 0
		;;
	esac
done

statscsv=$(dirname "$0")/stats.csv
collectors=${collectors:-bridge coverage datapath interface memory pmd-perf pmd-rxq vswitch}
metricsets=${metricsets:-base errors perf counters}
# shellcheck disable=SC2086
get_stats $collectors | sort | filter "$statscsv" "$metricsets"
exit 0
