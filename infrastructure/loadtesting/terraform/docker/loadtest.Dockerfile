FROM golang:1.21.5@sha256:4e2551bdfcc449e1363284ddba11e89607d88e915674b6f654a7a5bf47a83200
ARG TAG
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/cmd/osquery-perf/ && go build .

FROM golang:1.21.5@sha256:4e2551bdfcc449e1363284ddba11e89607d88e915674b6f654a7a5bf47a83200

COPY --from=0 /go/fleet/cmd/osquery-perf/osquery-perf /go/osquery-perf
