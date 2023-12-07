FROM golang:1.21.3@sha256:d0214956a9c50c300e430c1f6c0a820007ace238e5242c53762e61b344659e05
ARG TAG
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/cmd/osquery-perf/ && go build .

FROM golang:1.21.3@sha256:d0214956a9c50c300e430c1f6c0a820007ace238e5242c53762e61b344659e05

COPY --from=0 /go/fleet/cmd/osquery-perf/osquery-perf /go/osquery-perf
