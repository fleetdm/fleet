FROM golang:1.21.1@sha:0dff643e5bf836005eea93ad89e084a17681173e54dbaa9ec307fd776acab36e
ARG TAG
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/cmd/osquery-perf/ && go build .

FROM golang:1.21.1@sha:0dff643e5bf836005eea93ad89e084a17681173e54dbaa9ec307fd776acab36e
COPY --from=0 /go/fleet/cmd/osquery-perf/osquery-perf /go/osquery-perf
