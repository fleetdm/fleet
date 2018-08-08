FROM alpine
MAINTAINER Kolide Developers <engineering@kolide.co>

RUN apk --update add ca-certificates

COPY ./build/binary-bundle/linux/fleet ./build/binary-bundle/linux/fleetctl /usr/bin/

CMD ["fleet", "serve"]
