// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
)

var format = flag.String("l", "",
	"List all supported metrics in the specified format and exit.\n"+
		"Supported formats are: text, json, csv, tsv, markdown.")

func main() {
	flag.Parse()
	if *format != "" {
		lib.PrintMetrics(collectors.Collectors(), *format)
		os.Exit(0)
	}
	if err := config.Parse(); err != nil {
		// logging not initialized yet, directly write to stderr
		fmt.Fprintf(os.Stderr, "error: failed to parse config: %s\n", err)
		os.Exit(1)
	}
	if err := log.InitLogging(config.LogLevel()); err != nil {
		// logging not initialized yet, directly write to stderr
		fmt.Fprintf(os.Stderr, "error: failed to init log: %s\n", err)
		os.Exit(1)
	}

	log.Debugf("initializing collectors")

	registry := prometheus.NewRegistry()

	for _, c := range collectors.Collectors() {
		if lib.CollectorEnabled(c) {
			log.Infof("registering %T", c)

			if err := registry.Register(c); err != nil {
				log.Critf("collector: %s", err)
				os.Exit(1)
			}
		} else {
			log.Infof("%T not registered, metric set not enabled", c)
		}
	}

	handler := promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{
			ErrorLog:            log.PrometheusLogger(),
			ErrorHandling:       promhttp.ContinueOnError,
			MaxRequestsInFlight: 10,
			Timeout:             2 * time.Second,
			EnableOpenMetrics:   true,
		},
	)
	mux := http.NewServeMux()
	mux.Handle(config.HttpPath(), handler)

	server := &http.Server{
		Handler:           mux,
		ErrorLog:          log.ErrorLogger(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	listenAddresses := []string{config.HttpListen()}
	webConfigFile := config.WebConfigFile()
	systemdSocket := false

	flagConfig := &web.FlagConfig{
		WebListenAddresses: &listenAddresses,
		WebSystemdSocket:   &systemdSocket,
		WebConfigFile:      &webConfigFile,
	}

	log.Noticef("listening on %s%s", config.HttpListen(), config.HttpPath())

	if err := web.ListenAndServe(server, flagConfig, log.SlogLogger()); err != nil {
		log.Critf("listen: %s", err)
		os.Exit(1)
	}
}
