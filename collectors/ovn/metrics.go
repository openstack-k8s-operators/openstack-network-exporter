package ovn

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

var openvSwitch = map[string]lib.Metric{
	"ovn-remote-probe-interval": {
		Name:        "ovnc_remote_probe_interval",
		Description: "Maximum number of milliseconds of idle time on connection to the OVN SB DB before sending an inactivity probe message",
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_COUNTERS,
	},
	"ovn-openflow-probe-interval": {
		Name:        "ovnc_openflow_probe_interval",
		Description: "Maximum number of milliseconds of idle time on OpenFlow connection to the OVS bridge before sending an inactivity probe message",
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_COUNTERS,
	},
}

var openvSwitchBoolean = map[string]lib.Metric{
	"ovn-monitor-all": {
		Name:        "ovnc_monitor_all",
		Description: "Specifies if ovn-controller should monitor all records of tables in OVN SB DB. The value of 0 means it will conditionally monitor the records that are needed in the current chassis",
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_BASE,
	},
}

var openvSwitchLabels = map[string]lib.Metric{
	"ovn-encap-ip": {
		Name:        "ovnc_encap_ip",
		Description: "A metric with a constant '1' value labeled by ipadress that specifies the encapsulation ip address configured on that node",
		Labels:      []string{"encap_ip"},
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_BASE,
	},
	"ovn-remote": {
		Name:        "ovnc_sb_connection_method",
		Description: "A metric with a constant '1' value labeled by sb_connection_method that specifies the ovn-remote value configured on that node",
		Labels:      []string{"sb_connection_method"},
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_BASE,
	},
	"ovn-encap-type": {
		Name:        "ovnc_encap_type",
		Description: "A metric with a constant '1' value labeled by type that specifies the encapsulation type that a chassis should use to connect to this node",
		Labels:      []string{"encap_type"},
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_BASE,
	},
}

var bridgeMappings = lib.Metric{
	Name:        "ovnc_bridge_mappings",
	Description: "A metric with a constant '1' value labeled by mapping that specifies a list of key-value pairs that map a physical network name to a local ovs bridge that provides connectivity to that network.",
	Labels:      []string{"network", "bridge"},
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var ovnController = map[string]lib.Metric{
	"lflow_run": {
		Name:        "ovnc_lflow_run",
		Description: "Number of times ovn-controller has translated the Logical_Flow table in the OVN SB database into OpenFlow flow",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"rconn_sent": {
		Name:        "ovnc_rconn_sent",
		Description: "Specifies the number of messages that have been sent to the underlying virtual connection (unix, tcp, or ssl) to OpenFlow devices",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"rconn_queued": {
		Name:        "ovnc_rconn_queued",
		Description: "Specifies the number of messages that have been queued because it couldn’t be sent using the underlying virtual connection to OpenFlow devices",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rconn_discarded": {
		Name:        "ovnc_rconn_discarded",
		Description: "Specifies the number of messages that have been dropped because the send queue had to be flushed because of reconnection.",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rconn_overflow": {
		Name:        "ovnc_rconn_overflow",
		Description: "Specifies the number of messages that have been dropped because of the queue overflow",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"vconn_open": {
		Name:        "ovnc_vconn_open",
		Description: "Specifies the number of attempts to connect to an OpenFlow Device",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"vconn_sent": {
		Name:        "ovnc_vconn_sent",
		Description: "Specifies the number of messages sent to the OpenFlow Device",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"vconn_received": {
		Name:        "ovnc_vconn_received",
		Description: "Specifies the number of messages received from the OpenFlow Device",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"stream_open": {
		Name:        "ovnc_stream_open",
		Description: "Specifies the number of attempts to connect to a remote peer (active connection)",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"txn_success": {
		Name:        "ovnc_txn_success",
		Description: "Specifies the number of times the OVSDB transaction has successfully completed",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"txn_error": {
		Name:        "ovnc_txn_error",
		Description: "Specifies the number of times the OVSDB transaction has errored out",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"txn_uncommitted": {
		Name:        "ovnc_txn_uncommitted",
		Description: "Specifies the number of times the OVSDB transaction were uncommitted",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"txn_unchanged": {
		Name:        "ovnc_txn_unchanged",
		Description: "Specifies the number of times the OVSDB transaction resulted in no change to the database",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"txn_incomplete": {
		Name:        "ovnc_txn_incomplete",
		Description: "Specifies the number of times the OVSDB transaction did not complete and the client had to re-try",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"txn_aborted": {
		Name:        "ovnc_txn_aborted",
		Description: "Specifies the number of times the OVSDB transaction has been aborted",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"txn_try_again": {
		Name:        "ovnc_txn_try_again",
		Description: "Specifies the number of times the OVSDB transaction failed and the client had to re-try",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"pinctrl_total_pin_pkts": {
		Name:        "ovnc_pinctrl_total_pin_pkts",
		Description: "Specifies the number of times ovn-controller has handled the packet-ins from ovs-vswitchd",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"netlink_sent": {
		Name:        "ovnc_netlink_sent",
		Description: "Number of netlink message sent to the kernel",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"netlink_received": {
		Name:        "ovnc_netlink_received",
		Description: "Number of netlink messages received by the kernel",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},
	"netlink_recv_jumbo": {
		Name:        "ovnc_netlink_recv_jumbo",
		Description: "Number of netlink messages that were received from the kernel were more than the allocated buffer",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"netlink_overflow": {
		Name:        "ovnc_netlink_overflow",
		Description: "Netlink messages dropped by the daemon due to buffer overflow",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	packetInDrop: {
		Name:        "ovnc_" + packetInDrop,
		Description: "Specifies the number of times the ovn-controller has dropped the packet-ins from ovs-vswitchd due to resource constraints",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
}

var metrics = []*map[string]lib.Metric{
	&openvSwitch,
	&openvSwitchBoolean,
	&openvSwitchLabels,
	&ovnController,
}
