#!/bin/bash

c() {
   echo "+vm $*"
   "$@"
}
c sudo apt-get install openvswitch-switch-dpdk iperf3 -y
c sudo update-alternatives --set ovs-vswitchd /usr/lib/openvswitch-switch-dpdk/ovs-vswitchd-dpdk
c sudo /etc/init.d/openvswitch-switch start

c sudo sysctl -w vm.nr_hugepages=2048
c sudo modprobe vfio enable_unsafe_noiommu_mode=1
c sudo modprobe vfio-pci
c sudo ovs-vsctl set o . other_config:pmd-cpu-mask=0x06
c sudo ovs-vsctl set o . other_config:dpdk-extra="-a 0000:00:00.0 --iova-mode=pa"
c sudo ovs-vsctl set o . other_config:dpdk-init=true
c sudo /etc/init.d/openvswitch-switch restart

c sudo ovs-vsctl add-br br-phy-0 -- set bridge br-phy-0 datapath_type=netdev
c sudo ip link set br-phy-0 up

c sudo ovs-vsctl add-port br-phy-0 tap0 -- set interface tap0 type=dpdk options:dpdk-devargs=net_tap0 options:n_rxq=3
c sudo ovs-vsctl add-port br-phy-0 tap1 -- set interface tap1 type=dpdk options:dpdk-devargs=net_tap1 options:n_rxq=3

c sudo ip netns add ns_0
c sudo ip netns add ns_1

c sudo ip link set dtap0 netns ns_0
c sudo ip link set dtap1 netns ns_1

c sudo ip netns exec ns_0 ip addr add 10.10.10.10/24 dev dtap0
c sudo ip netns exec ns_0 ip link set dtap0 up
c sudo ip netns exec ns_1 ip addr add 10.10.10.11/24 dev dtap1
c sudo ip netns exec ns_1 ip link set dtap1 up

c sudo ip netns exec ns_1 ping -c 3 10.10.10.10
c sudo ip netns exec ns_0 iperf3 -s -B 10.10.10.10 -p 7575 -D --logfile /tmp/iperf3.txt --forceflush
c sleep 3
c sudo ip netns exec ns_1 iperf3 -c 10.10.10.10 -t 3 -p 7575
c sudo cat /tmp/iperf3.txt
c sudo killall iperf3
c sudo grep ERR /var/log/openvswitch/*

c sudo ovs-vsctl show
c sudo ovs-appctl dpif-netdev/pmd-rxq-show
c sudo ovs-appctl dpif-netdev/pmd-stats-show
c sudo ovs-vsctl list interface tap0
c sudo ovs-vsctl list interface tap1
c sudo ovs-vsctl list interface br-phy-0
c sudo ip a
for ns in $(ip netns ls | awk '{print $1}');do c sudo ip netns exec "${ns}" ip a;done
