FROM debian:13.4@sha256:e2d08da6f42ef4b09b165d55528a12727aeed8240dc9edf888e3ec07e10ef9da

COPY fleet-osquery_amd64.deb /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN apt-get update && apt-get install -y ca-certificates

RUN dpkg --install /fleet-osquery_amd64.deb

ENTRYPOINT ["/run-fleetd.sh"]
