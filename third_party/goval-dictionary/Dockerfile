FROM golang:alpine as builder

RUN apk add --no-cache \
        git \
        make \
        gcc \
        musl-dev

ENV REPOSITORY github.com/vulsio/goval-dictionary
COPY . $GOPATH/src/$REPOSITORY
RUN cd $GOPATH/src/$REPOSITORY && make install


FROM alpine:3.22

LABEL maintainer sadayuki-matsuno

ENV LOGDIR /var/log/goval-dictionary
ENV WORKDIR /goval-dictionary

RUN apk add --no-cache ca-certificates \
    && mkdir -p $WORKDIR $LOGDIR

COPY --from=builder /go/bin/goval-dictionary /usr/local/bin/

VOLUME ["$WORKDIR", "$LOGDIR"]
WORKDIR $WORKDIR
ENV PWD $WORKDIR

ENTRYPOINT ["goval-dictionary"]
CMD ["--help"]
