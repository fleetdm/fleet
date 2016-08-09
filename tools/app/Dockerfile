FROM golang:1.6.3-wheezy
MAINTAINER engineering@kolide.co

RUN mkdir -p /go/src/app
WORKDIR /go/src/app
COPY . /go/src/app

# Download and install any required third party dependencies into the container.
RUN go get github.com/tools/godep
RUN godep restore
RUN go build -o /go/src/app/kolide

CMD ./kolide serve
