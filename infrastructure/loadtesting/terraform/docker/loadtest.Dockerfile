FROM golang:1.21.6@sha256:5c7c2c9f1a930f937a539ff66587b6947890079470921d62ef1a6ed24395b4b3
ARG TAG
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/cmd/osquery-perf/ && go build .

FROM golang:1.21.6@sha256:5c7c2c9f1a930f937a539ff66587b6947890079470921d62ef1a6ed24395b4b3

COPY --from=0 /go/fleet/cmd/osquery-perf/osquery-perf /go/osquery-perf
