FROM ubuntu:20.04@sha256:8feb4d8ca5354def3d8fce243717141ce31e2c428701f6682bd2fafe15388214

COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN apt-get update && apt-get install -y ca-certificates

RUN --mount=type=bind,source=fleet-osquery_amd64.deb,target=/tmp/fleet-osquery_amd64.deb \
    dpkg --install /tmp/fleet-osquery_amd64.deb

ENTRYPOINT ["/run-fleetd.sh"]
