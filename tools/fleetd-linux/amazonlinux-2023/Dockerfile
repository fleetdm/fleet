FROM amazonlinux:2023

COPY fleet-osquery_amd64.rpm /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

# findutils for xargs
RUN dnf install --assumeyes findutils ca-certificates

RUN dnf install --assumeyes /fleet-osquery_amd64.rpm

ENTRYPOINT ["/run-fleetd.sh"]
