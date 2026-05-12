FROM golang:1.26.2-alpine3.23@sha256:80fbb8f9b2fa541a7d34378f1ad10f4f1c433817c4ed39ddb3e2f3ec3e961271
ARG TAG
RUN go install github.com/fleetdm/fleet/v4/tools/mdm/migration/mdmproxy@${TAG}

FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11
LABEL maintainer="Fleet Developers"

RUN apk update && apk add --no-cache tini
COPY --from=0 /go/bin/mdmproxy /usr/bin/mdmproxy
ADD --chmod=0755 ./entrypoint.sh /usr/bin/entrypoint.sh

# Create mdmproxy group and user
RUN addgroup -S mdmproxy && adduser -S mdmproxy -G mdmproxy
USER mdmproxy

ENTRYPOINT ["/sbin/tini", "/usr/bin/entrypoint.sh"]
