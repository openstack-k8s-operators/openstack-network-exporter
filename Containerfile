# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Robin Jarry

FROM registry.access.redhat.com/ubi9/ubi AS build
ADD . /src
RUN dnf install -y --nodocs --setopt=install_weak_deps=0 go
RUN cd /src && go generate ./... && go build -trimpath -o dataplane-node-exporter

FROM registry.access.redhat.com/ubi9/ubi-minimal
COPY --from=build /src/etc/dataplane-node-exporter.yaml /etc/dataplane-node-exporter.yaml
COPY --from=build /src/dataplane-node-exporter /app/dataplane-node-exporter
RUN microdnf update -y && rm -rf /var/cache/yum
RUN microdnf install -y iproute && microdnf clean all
MAINTAINER Red Hat
EXPOSE 1981
CMD ["/app/dataplane-node-exporter"]
