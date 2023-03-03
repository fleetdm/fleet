FROM alpine:3.17.2@sha256:e2e16842c9b54d985bf1ef9242a313f36b856181f188de21313820e177002501
LABEL maintainer="Fleet Developers"

RUN apk --update add ca-certificates
RUN apk --no-cache add jq

# Create FleetDM group and user
RUN addgroup -S fleet && adduser -S fleet -G fleet

COPY ./build/binary-bundle/linux/fleet ./build/binary-bundle/linux/fleetctl /usr/bin/

USER fleet
CMD ["fleet", "serve"]
