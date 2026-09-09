FROM cachyos/cachyos:latest@sha256:f654824d870b4222f8d6c2d4f57ecc6bd27c416123b78f103779044a10bcc0f0

COPY fleet-osquery_amd64.pkg.tar.zst /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN pacman -Sy --noconfirm --needed ca-certificates

RUN pacman -U --noconfirm /fleet-osquery_amd64.pkg.tar.zst

ENTRYPOINT ["/run-fleetd.sh"]
