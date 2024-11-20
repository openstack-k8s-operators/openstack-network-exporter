package lib

import (
	"fmt"
	"strings"

	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector interface {
	prometheus.Collector

	Name() string
	Metrics() []Metric
}

type Metric struct {
	Name        string
	Description string
	Labels      []string
	ValueType   prometheus.ValueType
	Set         config.MetricSet
	desc        *prometheus.Desc
}

func (m *Metric) Desc() *prometheus.Desc {
	if m.desc == nil {
		m.desc = prometheus.NewDesc(m.Name, m.Description, m.Labels, nil)
	}
	return m.desc
}

func DescribeEnabledMetrics(c Collector, ch chan<- *prometheus.Desc) {
	for _, m := range c.Metrics() {
		if config.MetricSets().Has(m.Set) {
			log.Debugf("%T: enabling metric %s", c, m.Name)
			ch <- m.Desc()
		}
	}
}

func PrintMetrics(c Collector) {
	for _, m := range c.Metrics() {
		fmt.Printf("%s", m.Name)
		fmt.Printf(" collector=%s", c.Name())
		fmt.Printf(" set=%s", m.Set)
		fmt.Printf(" type=%s", strings.ToLower(m.ValueType.ToDTO().String()))
		fmt.Printf(" labels=%s", strings.Join(m.Labels, ","))
		fmt.Printf(" help=%q", m.Description)
		fmt.Printf("\n")
	}
}

func CollectorEnabled(c Collector) bool {
	collectors := config.Collectors()
	if len(collectors) == 0 {
		return true
	}
	for _, name := range collectors {
		if name == c.Name() {
			return true
		}
	}
	return false
}
