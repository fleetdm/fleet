FROM quay.io/centos/centos@sha256:3a3a88a9c8c47dc60b8af3be0779d74c2d3da11a32082ac1a4c329863c422df3
# Pinning version because without the pin it's failing with host not found for mirrorlist.centos.org.

COPY fleet-osquery_amd64.rpm /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN dnf install --assumeyes /fleet-osquery_amd64.rpm

ENTRYPOINT ["/run-fleetd.sh"]
