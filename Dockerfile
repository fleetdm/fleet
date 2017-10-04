FROM alpine:3.4
MAINTAINER Kolide Developers <engineering@kolide.co>

RUN apk --update add \
    ca-certificates

COPY ./build/fleet /fleet

CMD ["/fleet", "serve"]
