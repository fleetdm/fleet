FROM amazonlinux:2023@sha256:1b091808a98a0a1b0570073421866f5a76adfa43e8d7a42b2f5dd0c60053a567

COPY fleet-osquery_amd64.rpm /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

# For xargs
RUN dnf install --assumeyes findutils

RUN dnf install --assumeyes /fleet-osquery_amd64.rpm

ENTRYPOINT ["/run-fleetd.sh"]
