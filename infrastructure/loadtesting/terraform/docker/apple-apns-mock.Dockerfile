FROM golang:1.27.3-alpine3.23@sha256:895e5feab11a4b3774d24ee3c9579028456b82ad37f42a9eb6f82adaf776a201
ARG TAG
RUN apk add git
RUN git clone -b "$TAG" --depth=1 --no-tags --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git
RUN cd /go/fleet && go build -o /go/bin/apple-apns-mock ./cmd/apple-apns-mock

FROM alpine:3.23.8@sha256:5c216c9767869f6ec7c4718977601d52f087c5c7bf2fdfb198e403f6b48ec40d
LABEL maintainer="Fleet Developers"

RUN addgroup -S apple-apns-mock && adduser -S apple-apns-mock -G apple-apns-mock

COPY --from=0 /go/bin/apple-apns-mock /go/apple-apns-mock

WORKDIR /go
USER apple-apns-mock

ENTRYPOINT ["/go/apple-apns-mock"]
