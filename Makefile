# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Robin Jarry

version = $(shell git describe --long --abbrev=12 --tags --dirty 2>/dev/null || echo v0.0.2)
src = $(shell find * -type f -name '*.go') go.mod go.sum

# Image URL to use all building/pushing image targets
DEFAULT_IMG ?= quay.io/openstack-k8s-operators/openstack-network-exporter:$(version)
# Development: quay.io/user/openstack-network-exporter:latest
IMG ?= $(DEFAULT_IMG)

.PHONY: all
all: openstack-network-exporter

openstack-network-exporter: $(src) ovsdb/ovs/model.go
	go build -trimpath -o $@

.PHONY: generate
generate: ovsdb/ovs/model.go

ovsdb/ovs/model.go: ovsdb/ovs/schema.json
	go generate ./...

.PHONY: debug
debug: openstack-network-exporter.debug

openstack-network-exporter.debug: $(src) ovsdb/ovs/model.go
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
run: openstack-network-exporter cert.pem key.pem
	OPENSTACK_NETWORK_EXPORTER_YAML=etc/dev.yaml ./$<

.PHONY: container
container:
	podman build -t ${IMG} .

.PHONY: push
push:
	podman push ${IMG}

.PHONY: tag-release
tag-release:
	@cur_version=`sed -En 's/.* \|\| echo v([0-9\.]+)\>.*$$/\1/p' Makefile` && \
	next_version=`echo $$cur_version | awk -F. -v OFS=. '{$$(NF) += 1; print}'` && \
	read -rp "next version ($$next_version)? " n && \
	if [ -n "$$n" ]; then next_version="$$n"; fi && \
	set -xe && \
	sed -i "s/\<v$$cur_version\>/v$$next_version/" Makefile && \
	git commit -sm "openstack-network-exporter: release v$$next_version" -m "`git shortlog -sn v$$cur_version..`" Makefile && \
	git tag -sm "v$$next_version" "v$$next_version"

.PHONY: shellcheck
shellcheck: test/*.sh
	shellcheck -e SC2317 test/*.sh

.PHONY: test
test: openstack-network-exporter
	sudo bash -x ./test/config_test_environment.sh
	sudo bash -x ./test/run_tests.sh -r 5
