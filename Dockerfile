FROM alpine
MAINTAINER Fleet Developers <hello@fleetdm.com>

RUN apk --update add ca-certificates

# Create FleetDM group and user
RUN addgroup -S fleet && adduser -S fleet -G fleet

COPY ./build/binary-bundle/linux/fleet ./build/binary-bundle/linux/fleetctl /usr/bin/

### Setup logging directory ###
RUN mkdir /var/log/osquery && \
	chown root:root /var/log/osquery && \
	touch /var/log/osquery/status.log && \
	touch /var/log/osquery/result.log && \
	chown fleet:fleet /var/log/osquery/status.log && \
	chown fleet:fleet /var/log/osquery/status.log

USER fleet
CMD ["fleet", "serve"]
