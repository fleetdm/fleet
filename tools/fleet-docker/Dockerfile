FROM alpine:3.23.3@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659
LABEL maintainer="Fleet Developers"

RUN apk --update add ca-certificates
RUN apk --no-cache add jq

# Create fleet group and user
RUN addgroup -S fleet && adduser -S fleet -G fleet

USER fleet

COPY fleet /usr/bin/

CMD ["fleet", "serve"]
