FROM golang:1.26.6-alpine3.23@sha256:e57c41c1d5864341031181b0db34b9a537bb5773eb6428e4e5bdaea0f9135406
ARG TAG
RUN apk update && apk add --no-cache git
RUN git clone --branch "${TAG}" --depth=1 --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git /go/fleet \
    && cd /go/fleet/tools/mdm/migration/mdmproxy \
    && go build -o /go/fleet/tools/mdm/migration/mdmproxy/mdmproxy .

FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11
LABEL maintainer="Fleet Developers"

RUN apk --no-cache upgrade openssl libcrypto3 libssl3 \
    && apk add --no-cache tini
COPY --from=0 /go/fleet/tools/mdm/migration/mdmproxy/mdmproxy /usr/bin/mdmproxy
ADD --chmod=0755 ./entrypoint.sh /usr/bin/entrypoint.sh

# Create mdmproxy group and user
RUN addgroup -S mdmproxy && adduser -S mdmproxy -G mdmproxy
USER mdmproxy

ENTRYPOINT ["/sbin/tini", "/usr/bin/entrypoint.sh"]
