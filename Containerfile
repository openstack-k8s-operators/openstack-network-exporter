# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Robin Jarry

FROM registry.access.redhat.com/ubi9/ubi AS build
RUN dnf install -y --nodocs --setopt=install_weak_deps=0 go

FROM build AS builder
ADD . /src
RUN cd /src && go generate ./... && go build -trimpath -o dataplane-node-exporter

FROM registry.access.redhat.com/ubi9/ubi-minimal AS minim_plus
RUN microdnf update -y && rm -rf /var/cache/yum
RUN microdnf install -y iproute && microdnf clean all

FROM minim_plus AS final
COPY --from=builder /src/etc/dataplane-node-exporter.yaml /etc/dataplane-node-exporter.yaml
COPY --from=builder /src/dataplane-node-exporter /app/dataplane-node-exporter
MAINTAINER Red Hat
EXPOSE 1981
CMD ["/app/dataplane-node-exporter"]
