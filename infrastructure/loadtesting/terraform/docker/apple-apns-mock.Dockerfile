# Must be >= the `go` directive in the cloned repo's go.mod. The official Go
# images set GOTOOLCHAIN=local, so a lower version here does not silently
# download a newer toolchain -- it fails the build.
FROM golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc
ARG TAG
RUN apk add git
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git
RUN cd /go/fleet && go build -o /go/bin/apple-apns-mock ./cmd/apple-apns-mock

FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11
LABEL maintainer="Fleet Developers"

RUN addgroup -S apple-apns-mock && adduser -S apple-apns-mock -G apple-apns-mock

COPY --from=0 /go/bin/apple-apns-mock /go/apple-apns-mock

WORKDIR /go
USER apple-apns-mock

ENTRYPOINT ["/go/apple-apns-mock"]
