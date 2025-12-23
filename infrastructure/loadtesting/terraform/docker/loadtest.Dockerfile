FROM golang:1.25.5-alpine3.23@sha256:26111811bc967321e7b6f852e914d14bede324cd1accb7f81811929a6a57fea9
ARG TAG
RUN apk add git sqlite gcc musl-dev sqlite-dev
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git && cd /go/fleet/cmd/osquery-perf/ && go build .

# Generate software database from SQL file
RUN cd /go/fleet/cmd/osquery-perf/software-library && \
    ls -lh && \
    if [ ! -f software.sql ]; then \
        echo "ERROR: software.sql not found in software-library directory"; \
        exit 1; \
    fi && \
    echo "Generating software.db from software.sql..." && \
    rm -f software.db && \
    sqlite3 software.db < software.sql && \
    if [ ! -f software.db ]; then \
        echo "ERROR: Failed to generate software.db"; \
        exit 1; \
    fi && \
    echo "Validating database..." && \
    sqlite3 software.db "SELECT COUNT(*) FROM software;" && \
    echo "Successfully generated software.db ($(du -h software.db | cut -f1))"

FROM alpine:3.23.0@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375
LABEL maintainer="Fleet Developers"

# Create FleetDM group and user
RUN addgroup -S osquery-perf && adduser -S osquery-perf -G osquery-perf

COPY --from=0 /go/fleet/cmd/osquery-perf/osquery-perf /go/osquery-perf
COPY --from=0 /go/fleet/server/vulnerabilities/testdata/ /go/fleet/server/vulnerabilities/testdata/
# Copy software database (generated in builder stage)
COPY --from=0 /go/fleet/cmd/osquery-perf/software-library/ /go/software-library/
RUN apk update && apk upgrade && apk add --no-cache sqlite-libs
WORKDIR /go
USER osquery-perf
