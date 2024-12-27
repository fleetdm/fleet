FROM debian:12.8

COPY fleet-osquery_amd64.deb /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN dpkg --install /fleet-osquery_amd64.deb

ENTRYPOINT ["/run-fleetd.sh"]
