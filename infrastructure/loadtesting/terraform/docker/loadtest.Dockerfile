FROM golang:1.19.12@sha256:ddbb6cbbc88a5e8d802c3ab7dc717d7a0634401c030266f4ff0f1933806f2ed9
ARG TAG
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/cmd/osquery-perf/ && go build .

FROM golang:1.19.12@sha256:ddbb6cbbc88a5e8d802c3ab7dc717d7a0634401c030266f4ff0f1933806f2ed9
COPY --from=0 /go/fleet/cmd/osquery-perf/osquery-perf /go/osquery-perf
