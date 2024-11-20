// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package config

import (
	"fmt"
	"log/syslog"
	"os"
	"reflect"
	"strings"

	"github.com/openstack-k8s-operators/dataplane-node-exporter/log"
	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "/etc/dataplane-node-exporter.yaml"

type MetricSet uint

const (
	METRICS_NONE MetricSet = 0
	METRICS_BASE           = 1 << iota
	METRICS_ERRORS
	METRICS_PERF
	METRICS_COUNTERS
	METRICS_DEBUG
	METRICS_DEFAULT = METRICS_BASE | METRICS_ERRORS | METRICS_PERF | METRICS_COUNTERS
)

func (s MetricSet) Has(o MetricSet) bool {
	return (s & o) == o
}

func (s MetricSet) String() string {
	var names []string
	if s.Has(METRICS_BASE) {
		names = append(names, "base")
	}
	if s.Has(METRICS_ERRORS) {
		names = append(names, "errors")
	}
	if s.Has(METRICS_PERF) {
		names = append(names, "perf")
	}
	if s.Has(METRICS_COUNTERS) {
		names = append(names, "counters")
	}
	if s.Has(METRICS_DEBUG) {
		names = append(names, "debug")
	}
	return strings.Join(names, ",")
}

type user struct {
	Name     string
	Password string
}

type conf struct {
	HttpListen string            `yaml:"http-listen" env:"DATAPLANE_NODE_EXPORTER_HTTP_LISTEN"`
	HttpPath   string            `yaml:"http-path" env:"DATAPLANE_NODE_EXPORTER_HTTP_PATH"`
	TlsCert    string            `yaml:"tls-cert" env:"DATAPLANE_NODE_EXPORTER_TLS_CERT"`
	TlsKey     string            `yaml:"tls-key" env:"DATAPLANE_NODE_EXPORTER_TLS_KEY"`
	AuthUsers  []user            `yaml:"auth-users"`
	users      map[string]string `yaml:"-"`
	OvsRundir  string            `yaml:"ovs-rundir" env:"DATAPLANE_NODE_EXPORTER_OVS_RUNDIR"`
	LogLevel   string            `yaml:"log-level" env:"DATAPLANE_NODE_EXPORTER_LOG_LEVEL"`
	logLevel   syslog.Priority   `yaml:"-"`
	Collectors []string          `yaml:"collectors"`
	MetricSets []string          `yaml:"metric-sets"`
	metricSets MetricSet         `yaml:"-"`
}

var c = conf{
	HttpListen: ":1981",
	HttpPath:   "/metrics",
	OvsRundir:  "/run/openvswitch",
	LogLevel:   "notice",
	users:      make(map[string]string),
}

func HttpListen() string           { return c.HttpListen }
func HttpPath() string             { return c.HttpPath }
func TlsCert() string              { return c.TlsCert }
func TlsKey() string               { return c.TlsKey }
func OvsRundir() string            { return c.OvsRundir }
func Collectors() []string         { return c.Collectors }
func LogLevel() syslog.Priority    { return c.logLevel }
func AuthUsers() map[string]string { return c.users }
func MetricSets() MetricSet        { return c.metricSets }

func Parse() error {
	path, configInEnv := os.LookupEnv("DATAPLANE_NODE_EXPORTER_YAML")
	if !configInEnv {
		path = defaultConfigPath
	}

	// parse yaml config file
	if file, err := os.Open(path); err == nil {
		dec := yaml.NewDecoder(file)
		if err = dec.Decode(&c); err != nil {
			return err
		}
	} else if configInEnv {
		return err
	}

	// override with values from environment
	typ := reflect.TypeOf(c)
	val := reflect.ValueOf(c)
	for i := 0; i < typ.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)
		env, found := fieldType.Tag.Lookup("env")
		if !found {
			continue
		}
		envValue, found := os.LookupEnv(env)
		if !found {
			continue
		}
		fieldVal.SetString(envValue)
	}

	// parse complex values
	for _, user := range c.AuthUsers {
		c.users[user.Name] = user.Password
	}
	if prio, err := log.ParseLogLevel(c.LogLevel); err != nil {
		return err
	} else {
		c.logLevel = prio
	}
	if sets, err := ParseMetricSets(c.MetricSets); err != nil {
		return err
	} else {
		c.metricSets = sets
	}

	return nil
}

func ParseMetricSets(names []string) (MetricSet, error) {
	var sets MetricSet

	if len(names) == 0 {
		sets = METRICS_DEFAULT
	} else {
		for _, name := range names {
			switch strings.ToLower(name) {
			case "base":
				sets |= METRICS_BASE
			case "errors":
				sets |= METRICS_ERRORS
			case "perf":
				sets |= METRICS_PERF
			case "counters":
				sets |= METRICS_COUNTERS
			case "debug":
				sets |= METRICS_DEBUG
			default:
				return sets, fmt.Errorf("invalid metric set name: %q", name)
			}
		}
	}

	return sets, nil
}
