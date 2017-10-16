FROM alpine
MAINTAINER Kolide Developers <engineering@kolide.co>

RUN apk --update add ca-certificates

COPY ./build/linux/fleet /usr/bin/fleet

CMD ["fleet", "serve"]
