FROM ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852

COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN apt-get update && apt-get install -y ca-certificates

RUN --mount=type=bind,source=fleet-osquery_amd64.deb,target=/tmp/fleet-osquery_amd64.deb \
    dpkg --install /tmp/fleet-osquery_amd64.deb

ENTRYPOINT ["/run-fleetd.sh"]
