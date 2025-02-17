#!/bin/bash

apt-get install openvswitch-switch-dpdk iperf3 socat -y
update-alternatives --set ovs-vswitchd \
	/usr/lib/openvswitch-switch-dpdk/ovs-vswitchd-dpdk
systemctl restart openvswitch-switch

ovs-vsctl set o . other_config:pmd-cpu-mask=0x06
ovs-vsctl set o . other_config:dpdk-extra="-a 0000:00:00.0 --no-huge --legacy-mem -m 2048"
ovs-vsctl set o . other_config:dpdk-init=true
systemctl restart openvswitch-switch

ovs-vsctl add-br br-phy-0 -- set bridge br-phy-0 datapath_type=netdev

for i in 0 1; do
	tap="dtap$i"
	ns="ns_$i"
	ovs-vsctl add-port br-phy-0 "$tap" -- set interface "$tap" type=dpdk \
		options:dpdk-devargs=net_tap$i,iface=$tap options:n_rxq=3
	ip netns add "$ns"
	ip link set "$tap" netns "$ns"
	ip -n "$ns" link set lo up
	ip -n "$ns" link set "$tap" up
	ip -n "$ns" addr add 10.10.10.1$i/24 dev "$tap"
	ip -n "$ns" addr show
done

ip netns exec ns_1 ping -i0.01 -c 3 10.10.10.10
ip netns exec ns_0 iperf3 -s -B 10.10.10.10 -p 7575 -D \
	--logfile /tmp/iperf3.txt --forceflush
ip netns exec ns_1 socat FILE:/dev/null TCP4-CONNECT:10.10.10.10:7575,retry=10
ip netns exec ns_1 iperf3 -c 10.10.10.10 -t 3 -p 7575
cat /tmp/iperf3.txt
killall iperf3

grep ERR /var/log/openvswitch/* || true
ovs-vsctl show
ovs-appctl dpif-netdev/pmd-rxq-show
ovs-appctl dpif-netdev/pmd-stats-show
ovs-vsctl list interface dtap0
ovs-vsctl list interface dtap1
ovs-vsctl list interface br-phy-0
ip addr show

test_dir=$(dirname "$0")
"${test_dir}"/../openstack-network-exporter -l csv > "${test_dir}"/stats.csv
