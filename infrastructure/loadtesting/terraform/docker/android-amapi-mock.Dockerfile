FROM golang:1.27.4-alpine3.23@sha256:146e3b9c7c1e2ec3d3f5348c8a686151a75e1976e50efac9fc022ebe446a9a82
ARG TAG
RUN apk add git
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git
RUN cd /go/fleet && go build -o /go/bin/android-amapi-mock ./cmd/android-amapi-mock

FROM alpine:3.23.8@sha256:5c216c9767869f6ec7c4718977601d52f087c5c7bf2fdfb198e403f6b48ec40d
LABEL maintainer="Fleet Developers"

RUN addgroup -S android-amapi-mock && adduser -S android-amapi-mock -G android-amapi-mock

COPY --from=0 /go/bin/android-amapi-mock /go/android-amapi-mock

WORKDIR /go
USER android-amapi-mock

ENTRYPOINT ["/go/android-amapi-mock"]
