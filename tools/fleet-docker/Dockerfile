FROM alpine:3.23.7@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63
ARG FLEET_VERSION=unknown
LABEL maintainer="Fleet Developers"
LABEL org.opencontainers.image.version="${FLEET_VERSION}"
LABEL org.opencontainers.image.vendor="Fleet Device Management Inc."
LABEL org.opencontainers.image.title="fleet"
LABEL org.opencontainers.image.source="https://github.com/fleetdm/fleet"
LABEL org.opencontainers.image.url="https://fleetdm.com"
LABEL org.opencontainers.image.documentation="https://fleetdm.com/docs"

RUN apk --update add ca-certificates
RUN apk --no-cache upgrade openssl libcrypto3 libssl3

# Create fleet group and user
RUN addgroup -S fleet && adduser -S fleet -G fleet

USER fleet

COPY fleet /usr/bin/

CMD ["fleet", "serve"]
