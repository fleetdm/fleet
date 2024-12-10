FROM redhat/ubi9:9.5@sha256:38791b293262ac2169eca2717e68e626a047d2b89fbd1da544db24ed0204efeb

COPY fleet-osquery_amd64.rpm /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN dnf install --assumeyes /fleet-osquery_amd64.rpm

ENTRYPOINT ["/run-fleetd.sh"]
