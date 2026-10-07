# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Robin Jarry

FROM registry.access.redhat.com/ubi9/ubi@sha256:5858f9ace07316e3b12caab62f6c2481a5030bb6bafdca5a9ea324c321ef36df AS build_base
RUN dnf install -y --nodocs --setopt=install_weak_deps=0 go

FROM build_base AS build
ADD . /src
RUN cd /src && go generate ./... && go build -trimpath -o openstack-network-exporter

FROM registry.access.redhat.com/ubi9/ubi-minimal@sha256:5ed244b62bbf4095080144d9d35eb8fcd3d39a9801f94aadd63b9d10978a01ae AS ubi_minimal
RUN microdnf update -y && microdnf clean all && rm -rf /var/cache/dnf
RUN microdnf install -y iproute && microdnf clean all && rm -rf /var/cache/dnf

FROM ubi_minimal
COPY --from=build /src/etc/openstack-network-exporter.yaml /etc/openstack-network-exporter.yaml
COPY --from=build /src/openstack-network-exporter /app/openstack-network-exporter

MAINTAINER Red Hat
EXPOSE 1981
CMD ["/app/openstack-network-exporter"]
