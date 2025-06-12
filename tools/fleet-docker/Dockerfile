FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c
LABEL maintainer="Fleet Developers"

RUN apk --update add ca-certificates
RUN apk --no-cache add jq

# Create fleet group and user
RUN addgroup -S fleet && adduser -S fleet -G fleet

USER fleet

COPY fleet /usr/bin/

CMD ["fleet", "serve"]
