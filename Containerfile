# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Robin Jarry

FROM registry.access.redhat.com/ubi10/ubi AS build_base
RUN dnf install -y --nodocs --setopt=install_weak_deps=0 go

FROM build_base AS build
ADD . /src
RUN cd /src && go generate ./... && go build -trimpath -o openstack-network-exporter

FROM registry.access.redhat.com/ubi10/ubi-minimal AS ubi_minimal
RUN microdnf update -y && microdnf clean all && rm -rf /var/cache/dnf
RUN microdnf install -y iproute && microdnf clean all && rm -rf /var/cache/dnf

FROM ubi_minimal
COPY --from=build /src/etc/openstack-network-exporter.yaml /etc/openstack-network-exporter.yaml
COPY --from=build /src/openstack-network-exporter /app/openstack-network-exporter

MAINTAINER Red Hat
EXPOSE 1981
CMD ["/app/openstack-network-exporter"]
