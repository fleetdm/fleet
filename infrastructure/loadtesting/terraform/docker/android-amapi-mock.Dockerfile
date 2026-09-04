FROM golang:1.27.1-alpine3.23@sha256:c8500dc1e6c8d8db60a2c6986bc591517c3be360ca448b9df43449dec430cc34
ARG TAG
RUN apk add git
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git
RUN cd /go/fleet && go build -o /go/bin/android-amapi-mock ./cmd/android-amapi-mock

FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11
LABEL maintainer="Fleet Developers"

RUN addgroup -S android-amapi-mock && adduser -S android-amapi-mock -G android-amapi-mock

COPY --from=0 /go/bin/android-amapi-mock /go/android-amapi-mock

WORKDIR /go
USER android-amapi-mock

ENTRYPOINT ["/go/android-amapi-mock"]
