FROM alpine
MAINTAINER Fleet Developers <engineering@fleetdm.com>

RUN apk --update add ca-certificates

COPY ./build/binary-bundle/linux/fleet ./build/binary-bundle/linux/fleetctl /usr/bin/

CMD ["fleet", "serve"]
