FROM golang:1.26.7-alpine3.23@sha256:b17af760035fc2f338eed92d448a6c67f2d45438844fc6c60678fa5f99e44b57

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
