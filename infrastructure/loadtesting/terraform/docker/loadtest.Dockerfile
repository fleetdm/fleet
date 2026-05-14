FROM golang:1.26.3-alpine3.23@sha256:f44b851aa23dfa219d18db6eab743203245429d355cb619cf96a2ffe2a84ba7a
ARG TAG
RUN apk add git sqlite gcc musl-dev sqlite-dev
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git
RUN go install github.com/fleetdm/fleet/v4/cmd/osquery-perf@${TAG}

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

FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11
LABEL maintainer="Fleet Developers"

# Create FleetDM group and user
RUN addgroup -S osquery-perf && adduser -S osquery-perf -G osquery-perf

COPY --from=0 /go/bin/osquery-perf /go/osquery-perf
COPY --from=0 /go/fleet/server/vulnerabilities/testdata/ /go/fleet/server/vulnerabilities/testdata/
# Copy software database (generated in builder stage)
COPY --from=0 /go/fleet/cmd/osquery-perf/software-library/ /go/software-library/
RUN apk update && apk upgrade && apk add --no-cache sqlite-libs
WORKDIR /go
USER osquery-perf
