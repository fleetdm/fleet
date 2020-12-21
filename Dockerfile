FROM alpine
MAINTAINER Fleet Developers <engineering@fleetdm.com>

RUN apk --update add ca-certificates

# Create FleetDM group and user
RUN addgroup -S fleetdm && adduser -S fleetdm -G fleetdm

COPY ./build/binary-bundle/linux/fleet ./build/binary-bundle/linux/fleetctl /usr/bin/

USER fleetdm
CMD ["fleet", "serve"]
