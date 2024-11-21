// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	var err error
	server := http.Server{Addr: config.HttpListen(), Handler: mux, ErrorLog: log.ErrorLogger()}

	if config.TlsCert() != "" && config.TlsKey() != "" {
		log.Noticef("listening on https://%s%s", config.HttpListen(), config.HttpPath())
		server.Handler = basicAuthHandler(mux)
		err = server.ListenAndServeTLS(config.TlsCert(), config.TlsKey())
	} else {
		log.Noticef("listening on http://%s%s", config.HttpListen(), config.HttpPath())
		err = server.ListenAndServe()
	}
	if err != nil {
		log.Critf("listen: %s", err)
		os.Exit(1)
	}
}

func basicAuthHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user, password, pass string
		var ok, authorized bool

		if user, pass, ok = r.BasicAuth(); ok {
			if password, ok = config.AuthUsers()[user]; ok && pass == password {
				authorized = true
			}
		}
		if authorized {
			handler.ServeHTTP(w, r)
		} else {
			w.Header().Add("WWW-Authenticate", `Basic realm="ovs-node-exporter"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	})
}
