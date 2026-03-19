FROM golang:1.26.1-alpine3.23@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039
ARG TAG
RUN apk update && apk add --no-cache git
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/tools/mdm/migration/mdmproxy && go build .

FROM alpine:3.23.3@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659
LABEL maintainer="Fleet Developers"

RUN apk update && apk add --no-cache tini
COPY --from=0 /go/fleet/tools/mdm/migration/mdmproxy/mdmproxy /usr/bin/mdmproxy
ADD --chmod=0755 ./entrypoint.sh /usr/bin/entrypoint.sh

# Create mdmproxy group and user
RUN addgroup -S mdmproxy && adduser -S mdmproxy -G mdmproxy
USER mdmproxy

ENTRYPOINT ["/sbin/tini", "/usr/bin/entrypoint.sh"]
