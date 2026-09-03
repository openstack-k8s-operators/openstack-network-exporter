# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Robin Jarry

FROM registry.access.redhat.com/ubi10/ubi@sha256:87aaae2d47f11416cfb24a581a17f9875a3e59e9ed3e03e1b264887f653e4471 AS build_base
RUN dnf install -y --nodocs --setopt=install_weak_deps=0 go

FROM build_base AS build
ADD . /src
RUN cd /src && go generate ./... && go build -trimpath -o openstack-network-exporter

FROM registry.access.redhat.com/ubi10/ubi-minimal@sha256:d801168f5e8b108586c27a4fd5c92e3c1e8d061084383713926e2ca61b8b6c64 AS ubi_minimal
RUN microdnf update -y && microdnf clean all && rm -rf /var/cache/dnf
RUN microdnf install -y iproute && microdnf clean all && rm -rf /var/cache/dnf

FROM ubi_minimal
COPY --from=build /src/etc/openstack-network-exporter.yaml /etc/openstack-network-exporter.yaml
COPY --from=build /src/openstack-network-exporter /app/openstack-network-exporter

MAINTAINER Red Hat
EXPOSE 1981
CMD ["/app/openstack-network-exporter"]
