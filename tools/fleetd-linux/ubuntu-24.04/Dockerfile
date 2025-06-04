FROM ubuntu:24.04

COPY fleet-osquery_amd64.deb /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN apt-get update && apt-get install -y ca-certificates

RUN dpkg --install /fleet-osquery_amd64.deb

ENTRYPOINT ["/run-fleetd.sh"]
