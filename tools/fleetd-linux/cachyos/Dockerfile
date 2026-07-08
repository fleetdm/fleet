FROM cachyos/cachyos:latest

COPY fleet-osquery_amd64.pkg.tar.zst /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN pacman -Sy --noconfirm --needed ca-certificates

RUN pacman -U --noconfirm /fleet-osquery_amd64.pkg.tar.zst

ENTRYPOINT ["/run-fleetd.sh"]
