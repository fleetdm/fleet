FROM alpine:3.21@sha256:2c43f33bd1502ec7818bce9eea60e062d04eeadc4aa31cad9dabecb1e48b647b
LABEL maintainer="Fleet Developers"

RUN apk --update add ca-certificates
RUN apk --no-cache add jq

# Create fleet group and user
RUN addgroup -S fleet && adduser -S fleet -G fleet

USER fleet

COPY fleet /usr/bin/

CMD ["fleet", "serve"]
