FROM golang:1.21.7@sha256:549dd88a1a53715f177b41ab5fee25f7a376a6bb5322ac7abe263480d9554021
ARG TAG
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/cmd/osquery-perf/ && go build .

FROM golang:1.21.7@sha256:549dd88a1a53715f177b41ab5fee25f7a376a6bb5322ac7abe263480d9554021

COPY --from=0 /go/fleet/cmd/osquery-perf/osquery-perf /go/osquery-perf
