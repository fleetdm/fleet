FROM fedora:43@sha256:49b5f0ab05516b1a8168c79f295da65e8d8db1cae275a4bec007c0b88cd19ffb

COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN dnf install --assumeyes ca-certificates

RUN --mount=type=bind,source=fleet-osquery_amd64.rpm,target=/tmp/fleet-osquery_amd64.rpm \
    dnf install --assumeyes /tmp/fleet-osquery_amd64.rpm

ENTRYPOINT ["/run-fleetd.sh"]
