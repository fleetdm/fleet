FROM golang:1.27.4-alpine3.23@sha256:146e3b9c7c1e2ec3d3f5348c8a686151a75e1976e50efac9fc022ebe446a9a82
ARG TAG
RUN apk add git
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git
# fleetctl from the same tag runs the GitOps applies, as a customer's CI would with the server's version.
RUN cd /go/fleet && go build -o /go/bin/driver ./tools/loadtest/qa/driver && go build -o /go/bin/fleetctl ./cmd/fleetctl

FROM alpine:3.23.8@sha256:5c216c9767869f6ec7c4718977601d52f087c5c7bf2fdfb198e403f6b48ec40d
LABEL maintainer="Fleet Developers"

# aws-cli fetches the private GitOps bundle from S3 at start.
RUN apk update && apk upgrade && apk add --no-cache ca-certificates aws-cli
RUN addgroup -S driver && adduser -S driver -G driver

COPY --from=0 /go/bin/driver /go/bin/fleetctl /usr/local/bin/
COPY --from=0 /go/fleet/tools/loadtest/qa/scenarios/ /scenarios/
COPY activity-driver-entrypoint.sh /usr/local/bin/activity-driver-entrypoint.sh

USER driver
WORKDIR /home/driver
ENTRYPOINT ["/usr/local/bin/activity-driver-entrypoint.sh"]
