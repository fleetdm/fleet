FROM golang:1.8-alpine
MAINTAINER Kolide Developers <engineering@kolide.co>

# path to compiled version of node-sass for alpine
# https://github.com/sass/node-sass/issues/1589
ENV SASS_BINARY_PATH=/linux-x64-48_binding.node

RUN apk add --update \
        git \
        nodejs \
		bash \
		alpine-sdk \
		musl-dev \
		openssl

RUN \
   go get -u -v github.com/Masterminds/glide && \
   go get -u -v gopkg.in/alecthomas/gometalinter.v1

COPY rootfs /

VOLUME   /go/src/github.com/kolide/kolide
WORKDIR  /go/src/github.com/kolide/kolide

EXPOSE 8080

ENTRYPOINT ["/builder.sh"]

