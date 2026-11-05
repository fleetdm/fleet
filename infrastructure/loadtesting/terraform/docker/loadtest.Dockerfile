FROM golang:1.27.3-alpine3.23@sha256:895e5feab11a4b3774d24ee3c9579028456b82ad37f42a9eb6f82adaf776a201
ARG TAG
RUN apk add git sqlite gcc musl-dev sqlite-dev
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git
# Build from the clone instead of `go install ...@${TAG}`: installing by module path fetches the module zip,
# and the Fleet monorepo exceeds Go's hardcoded 500 MB module-zip limit.
RUN cd /go/fleet && go build -o /go/bin/osquery-perf ./cmd/osquery-perf

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

FROM alpine:3.23.7@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63
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
