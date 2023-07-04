FROM golang:1.19.10
ARG TAG
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/cmd/osquery-perf/ && go build .

FROM golang:1.19.10
COPY --from=0 /go/fleet/cmd/osquery-perf/osquery-perf /go/osquery-perf
