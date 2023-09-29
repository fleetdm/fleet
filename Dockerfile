FROM alpine:3.18.4@sha256:eece025e432126ce23f223450a0326fbebde39cdf496a85d8c016293fc851978
LABEL maintainer="Fleet Developers"

RUN apk --update add ca-certificates
RUN apk --no-cache add jq

# Create FleetDM group and user
RUN addgroup -S fleet && adduser -S fleet -G fleet

COPY ./build/binary-bundle/linux/fleet ./build/binary-bundle/linux/fleetctl /usr/bin/

USER fleet
CMD ["fleet", "serve"]
