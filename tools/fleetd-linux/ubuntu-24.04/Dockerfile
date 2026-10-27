FROM ubuntu:24.04

COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN apt-get update && apt-get install -y ca-certificates

RUN --mount=type=bind,source=fleet-osquery_amd64.deb,target=/tmp/fleet-osquery_amd64.deb \
    dpkg --install /tmp/fleet-osquery_amd64.deb

ENTRYPOINT ["/run-fleetd.sh"]
