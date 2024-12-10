FROM fedora:41@sha256:3ec60eb34fa1a095c0c34dd37cead9fd38afb62612d43892fcf1d3425c32bc1e

COPY fleet-osquery_amd64.rpm /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN dnf install --assumeyes /fleet-osquery_amd64.rpm

ENTRYPOINT ["/run-fleetd.sh"]
