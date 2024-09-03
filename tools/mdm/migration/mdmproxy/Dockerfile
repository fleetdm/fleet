FROM golang:1.22.6-alpine3.20@sha256:1a478681b671001b7f029f94b5016aed984a23ad99c707f6a0ab6563860ae2f3
ARG TAG
RUN apk update && apk add --no-cache git
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/tools/mdm/migration/mdmproxy && go build .

FROM alpine:3.20.1@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0
LABEL maintainer="Fleet Developers"

RUN apk update && apk add --no-cache tini
COPY --from=0 /go/fleet/tools/mdm/migration/mdmproxy/mdmproxy /usr/bin/mdmproxy
ADD --chmod=0755 ./entrypoint.sh /usr/bin/entrypoint.sh

# Create mdmproxy group and user
RUN addgroup -S mdmproxy && adduser -S mdmproxy -G mdmproxy
USER mdmproxy

ENTRYPOINT ["/sbin/tini", "/usr/bin/entrypoint.sh"]
