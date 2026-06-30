FROM golang:1.26.4-alpine3.23@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f
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
