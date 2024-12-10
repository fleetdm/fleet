FROM ubuntu:24.04@sha256:80dd3c3b9c6cecb9f1667e9290b3bc61b78c2678c02cbdae5f0fea92cc6734ab

COPY fleet-osquery_amd64.deb /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN apt-get update && apt-get install -y ca-certificates

RUN dpkg --install /fleet-osquery_amd64.deb

ENTRYPOINT ["/run-fleetd.sh"]
