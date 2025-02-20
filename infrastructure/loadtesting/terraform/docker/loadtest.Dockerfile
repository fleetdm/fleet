FROM golang:1.23.4-alpine3.21@sha256:052793ea3143a235a5b2d815ccead8910cfe547b36a1f4c8b070015b89da5eab
ARG TAG
RUN apk add git
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/cmd/osquery-perf/ && go build .

FROM alpine:3.21@sha256:2c43f33bd1502ec7818bce9eea60e062d04eeadc4aa31cad9dabecb1e48b647b
LABEL maintainer="Fleet Developers"

# Create FleetDM group and user
RUN addgroup -S osquery-perf && adduser -S osquery-perf -G osquery-perf

COPY --from=0 /go/fleet/cmd/osquery-perf/osquery-perf /go/osquery-perf
COPY --from=0 /go/fleet/server/vulnerabilities/testdata/ /go/fleet/server/vulnerabilities/testdata/
RUN set -eux; \
        apk update; \
        apk upgrade

USER osquery-perf
