FROM debian:12.8@sha256:b877a1a3fdf02469440f1768cf69c9771338a875b7add5e80c45b756c92ac20a

COPY fleet-osquery_amd64.deb /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN dpkg --install /fleet-osquery_amd64.deb

ENTRYPOINT ["/run-fleetd.sh"]
