# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Robin Jarry

version = $(shell git describe --long --abbrev=12 --tags --dirty 2>/dev/null || echo 0.1)
src = $(shell find * -type f -name '*.go') go.mod go.sum

.PHONY: all
all: dataplane-node-exporter

dataplane-node-exporter: $(src) ovsdb/ovs/model.go
	go build -trimpath -o $@

.PHONY: generate
generate: ovsdb/ovs/model.go

ovsdb/ovs/model.go: ovsdb/ovs/schema.json
	go generate ./...

.PHONY: debug
debug: dataplane-node-exporter.debug

dataplane-node-exporter.debug: $(src) ovsdb/ovs/model.go
	go build -gcflags=all="-N -l" -o $@

.PHONY: format
format:
	gofmt -w .

.PHONY: lint
lint: ovsdb/ovs/model.go
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0 run

REVISION_RANGE ?= origin/main..

.PHONY: check-commits
check-commits:
	$Q ./check-commits $(REVISION_RANGE)

.PHONY: cert
cert: cert.pem key.pem

cert.pem key.pem:
	openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem \
		-sha256 -days 3650 -nodes \
		-subj "/C=XX/ST=StateName/L=CityName/O=CompanyName/OU=CompanySectionName/CN=CommonNameOrHostname"

.PHONY: run
run: dataplane-node-exporter
	DATAPLANE_NODE_EXPORTER_YAML=etc/dev.yaml ./$<

.PHONY: container
container:
	podman build -t 'quay.io/openstack/dataplane-node-exporter:$(version)' .
