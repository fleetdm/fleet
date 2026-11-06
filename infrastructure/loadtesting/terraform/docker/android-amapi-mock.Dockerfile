FROM golang:1.27.3-alpine3.23@sha256:895e5feab11a4b3774d24ee3c9579028456b82ad37f42a9eb6f82adaf776a201
ARG TAG
RUN apk add git
RUN git clone -b $TAG --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git
RUN cd /go/fleet && go build -o /go/bin/android-amapi-mock ./cmd/android-amapi-mock

FROM alpine:3.23.7@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63
LABEL maintainer="Fleet Developers"

RUN addgroup -S android-amapi-mock && adduser -S android-amapi-mock -G android-amapi-mock

COPY --from=0 /go/bin/android-amapi-mock /go/android-amapi-mock

WORKDIR /go
USER android-amapi-mock

ENTRYPOINT ["/go/android-amapi-mock"]
