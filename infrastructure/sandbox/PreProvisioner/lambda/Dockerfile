FROM rust:latest@sha256:02a53e734724bef4a58d856c694f826aa9e7ea84353516b76d9a6d241e9da60e AS builder

ARG transporter_url=https://itunesconnect.apple.com/WebObjects/iTunesConnect.woa/ra/resources/download/public/Transporter__Linux/bin

RUN cargo install --version 0.16.0 apple-codesign \
  && curl -sSf $transporter_url -o transporter_install.sh \
  && sh transporter_install.sh --target transporter --accept --noexec

FROM golang:1.19.11-bullseye@sha256:0649acb9e15690c802857ceba11ee6542a50d54d7630ccc2dac72b828637b908

RUN apt-get update \
  && dpkg --add-architecture i386 \
  && apt update \
  && apt install -y --no-install-recommends ca-certificates cpio libxml2 wine wine32 libgtk-3-0 \
  && rm -rf /var/lib/apt/lists/*

# copy macOS dependencies
COPY --from=fleetdm/bomutils:latest /usr/bin/mkbom /usr/local/bin/xar /usr/bin/
COPY --from=fleetdm/bomutils:latest /usr/local/lib /usr/local/lib/
COPY --from=builder /transporter/itms /usr/local/
COPY --from=builder /usr/local/cargo/bin/rcodesign /usr/local/bin

# copy Windows dependencies
COPY --from=fleetdm/wix:latest /home/wine /home/wine

ENV FLEETCTL_NATIVE_TOOLING=1 WINEPREFIX=/home/wine/.wine WINEARCH=win32 PATH="/home/wine/bin:$PATH" WINEDEBUG=-all

RUN apt update; apt install -y curl openssl unzip
WORKDIR /build
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-extldflags '-static'"
RUN curl https://releases.hashicorp.com/terraform/1.1.8/terraform_1.1.8_linux_amd64.zip > terraform.zip
RUN unzip terraform.zip
RUN rm terraform.zip
RUN chmod 644 $(find . -type f)
RUN chmod 755 $(find . -type d)
RUN chmod 655 lambda terraform

#FROM scratch
#COPY --from=builder /build/lambda /build/terraform /
#COPY --from=builder /build/deploy_terraform /deploy_terraform
#COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/build/lambda"]
