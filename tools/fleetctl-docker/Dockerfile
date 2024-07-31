FROM rust:latest@sha256:4c45f61ebe054560190f232b7d883f174ff287e1a0972c8f6d7ab88da0188870 as builder

ARG transporter_url=https://itunesconnect.apple.com/WebObjects/iTunesConnect.woa/ra/resources/download/public/Transporter__Linux/bin

RUN cargo install --version 0.16.0 apple-codesign \
  && curl -sSf $transporter_url -o transporter_install.sh \
  && sh transporter_install.sh --target transporter --accept --noexec

FROM debian:stable-slim@sha256:f8bbfa052db81e5b8ac12e4a1d8310a85d1509d4d0d5579148059c0e8b717d4e

ARG binpath=build/binary-bundle/linux/fleetctl

RUN apt-get update \
  && dpkg --add-architecture i386 \
  && apt update \
  && apt upgrade -y \
  && apt install -y --no-install-recommends ca-certificates cpio libxml2 wine wine32 libgtk-3-0 libssl1.1 \
  && rm -rf /var/lib/apt/lists/* 

# copy macOS dependencies
COPY --from=fleetdm/bomutils:latest /usr/bin/mkbom /usr/local/bin/xar /usr/bin/
COPY --from=fleetdm/bomutils:latest /usr/local/lib /usr/local/lib/
COPY --from=builder /transporter/itms /usr/local/
COPY --from=builder /usr/local/cargo/bin/rcodesign /usr/local/bin

# copy Windows dependencies
COPY --from=fleetdm/wix:latest /home/wine /home/wine

# copy fleetctl
COPY ${binpath} /usr/bin/fleetctl

ENV FLEETCTL_NATIVE_TOOLING=1 WINEPREFIX=/home/wine/.wine WINEARCH=win32 PATH="/home/wine/bin:$PATH" WINEDEBUG=-all

ENTRYPOINT ["fleetctl"]
