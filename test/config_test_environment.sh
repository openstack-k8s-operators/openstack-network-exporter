#!/bin/bash

apt-get install openvswitch-switch-dpdk iperf3 -y
update-alternatives --set ovs-vswitchd \
/usr/lib/openvswitch-switch-dpdk/ovs-vswitchd-dpdk
/etc/init.d/openvswitch-switch start

sysctl -w vm.nr_hugepages=2048
modprobe vfio enable_unsafe_noiommu_mode=1
modprobe vfio-pci
ovs-vsctl set o . other_config:pmd-cpu-mask=0x06
ovs-vsctl set o . other_config:dpdk-extra="-a 0000:00:00.0 --iova-mode=pa"
ovs-vsctl set o . other_config:dpdk-init=true
/etc/init.d/openvswitch-switch restart

ovs-vsctl add-br br-phy-0 -- set bridge br-phy-0 datapath_type=netdev
ip link set br-phy-0 up

ovs-vsctl add-port br-phy-0 tap0 -- set interface tap0 type=dpdk \
options:dpdk-devargs=net_tap0 options:n_rxq=3
ovs-vsctl add-port br-phy-0 tap1 -- set interface tap1 type=dpdk \
options:dpdk-devargs=net_tap1 options:n_rxq=3

ip netns add ns_0
ip netns add ns_1

ip link set dtap0 netns ns_0
ip link set dtap1 netns ns_1

ip netns exec ns_0 ip addr add 10.10.10.10/24 dev dtap0
ip netns exec ns_0 ip link set dtap0 up
ip netns exec ns_1 ip addr add 10.10.10.11/24 dev dtap1
ip netns exec ns_1 ip link set dtap1 up

ip netns exec ns_1 ping -c 3 10.10.10.10
ip netns exec ns_0 iperf3 -s -B 10.10.10.10 -p 7575 -D \
--logfile /tmp/iperf3.txt --forceflush
sleep 3
ip netns exec ns_1 iperf3 -c 10.10.10.10 -t 3 -p 7575
cat /tmp/iperf3.txt
killall iperf3
grep ERR /var/log/openvswitch/*

ovs-vsctl show
ovs-appctl dpif-netdev/pmd-rxq-show
ovs-appctl dpif-netdev/pmd-stats-show
ovs-vsctl list interface tap0
ovs-vsctl list interface tap1
ovs-vsctl list interface br-phy-0
ip a
for ns in $(ip netns ls | awk '{print $1}');do ip netns exec "${ns}" ip a;done

test_dir=$(dirname "$0")
"${test_dir}"/../openstack-network-exporter -l csv > "${test_dir}"/stats.csv
