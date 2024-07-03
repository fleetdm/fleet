FROM alpine:3.20.1@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0
LABEL maintainer="Fleet Developers"

RUN apk --update add ca-certificates
RUN apk --no-cache add jq

# Create fleet group and user
RUN addgroup -S fleet && adduser -S fleet -G fleet

USER fleet

COPY fleet /usr/bin/
COPY fleetctl /usr/bin/

CMD ["fleet", "serve"]
