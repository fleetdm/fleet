FROM alpine
MAINTAINER Fleet Developers <hello@fleetdm.com>

RUN apk --update add ca-certificates

# Create fleet group and user
RUN addgroup -S fleet && adduser -S fleet -G fleet

USER fleet

COPY fleetctl /usr/bin/

ENTRYPOINT ["fleetctl"]
