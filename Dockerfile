FROM golang:1.6.3-wheezy
MAINTAINER engineering@kolide.co

RUN mkdir -p /app
WORKDIR /app
COPY . /app

# Download and install any required third party dependencies into the container.
RUN go-wrapper download
RUN go build -o kolide

CMD /app/kolide serve
