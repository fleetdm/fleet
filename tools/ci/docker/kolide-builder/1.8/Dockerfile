FROM golang:1.8-alpine
MAINTAINER Kolide Developers <engineering@kolide.co>

RUN apk add --update \
        git \
        nodejs \
		bash \
		alpine-sdk \
		musl-dev \
		openssl

RUN go get -u -v github.com/Masterminds/glide

RUN echo -e 'http://dl-cdn.alpinelinux.org/alpine/edge/main\nhttp://dl-cdn.alpinelinux.org/alpine/edge/community\nhttp://dl-cdn.alpinelinux.org/alpine/edge/testing' > /etc/apk/repositories && \
  apk add --no-cache yarn

# git must be explicitly instructed to follow redirects for compatibility with
# gopkg.in
RUN git config --global http.https://gopkg.in.followRedirects true

COPY rootfs /

VOLUME   /go/src/github.com/kolide/kolide
WORKDIR  /go/src/github.com/kolide/kolide

EXPOSE 8080

ENTRYPOINT ["/builder.sh"]
