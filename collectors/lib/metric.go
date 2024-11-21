package lib

import (
	"encoding/json"
	"fmt"
	"os"
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

var columns = []any{"METRIC", "COLLECTOR", "SET", "TYPE", "LABELS", "HELP"}

const (
	textFmt     = "%s collector=%s set=%s type=%s labels=%s help=%q\n"
	csvFmt      = "%s;%s;%s;%s;%s;%s;\n"
	tsvFmt      = "%s\t%s\t%s\t%s\t%s\t%s\n"
	markdownFmt = "| %s | %s | %s | %s | %s | %s |\n"
)

func PrintMetrics(collectors []Collector, format string) {
	var jsonList []map[string]any

	switch format {
	case "text", "json":
		break
	case "csv":
		fmt.Printf(csvFmt, columns...)
	case "tsv":
		fmt.Printf(tsvFmt, columns...)
	case "markdown":
		fmt.Printf(markdownFmt, columns...)
		separators := make([]any, 0, len(columns))
		for _, c := range columns {
			separators = append(separators, strings.Repeat("-", len(c.(string))))
		}
		fmt.Printf(markdownFmt, separators...)
	default:
		fmt.Fprintf(os.Stderr,
			"error: invalid format %q. "+
				"Supported formats are: text, csv, tsv, markdown.\n",
			format)
		os.Exit(1)
	}

	for _, c := range collectors {
		for _, m := range c.Metrics() {
			values := []any{
				m.Name,
				c.Name(),
				m.Set.String(),
				strings.ToLower(m.ValueType.ToDTO().String()),
				strings.Join(m.Labels, ","),
				m.Description,
			}
			switch format {
			case "text":
				fmt.Printf(textFmt, values...)
			case "json":
				jsonList = append(jsonList, map[string]any{
					"metric":    m.Name,
					"collector": c.Name(),
					"set":       m.Set.String(),
					"type":      strings.ToLower(m.ValueType.ToDTO().String()),
					"labels":    m.Labels,
					"help":      m.Description,
				})
			case "csv":
				fmt.Printf(csvFmt, values...)
			case "tsv":
				fmt.Printf(tsvFmt, values...)
			case "markdown":
				fmt.Printf(markdownFmt, values...)
			}
		}
	}
	if format == "json" {
		e := json.NewEncoder(os.Stdout)
		_ = e.Encode(jsonList)
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
