FROM golang:1.27.3-alpine3.23@sha256:895e5feab11a4b3774d24ee3c9579028456b82ad37f42a9eb6f82adaf776a201
ARG TAG
RUN apk update && apk add --no-cache git
RUN git clone --branch "${TAG}" --depth=1 --progress --no-recurse-submodules https://github.com/fleetdm/fleet.git /go/fleet \
    && cd /go/fleet/tools/mdm/migration/mdmproxy \
    && go build -o /go/fleet/tools/mdm/migration/mdmproxy/mdmproxy .

FROM alpine:3.23.7@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63
LABEL maintainer="Fleet Developers"

RUN apk --no-cache upgrade openssl libcrypto3 libssl3 \
    && apk add --no-cache tini
COPY --from=0 /go/fleet/tools/mdm/migration/mdmproxy/mdmproxy /usr/bin/mdmproxy
ADD --chmod=0755 ./entrypoint.sh /usr/bin/entrypoint.sh

# Create mdmproxy group and user
RUN addgroup -S mdmproxy && adduser -S mdmproxy -G mdmproxy
USER mdmproxy

ENTRYPOINT ["/sbin/tini", "/usr/bin/entrypoint.sh"]
