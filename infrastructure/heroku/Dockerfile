# Use a base image that includes necessary tools
FROM alpine:3.18.2@sha256:82d1e9d7ed48a7523bdebc18cf6290bdb97b82302a8a9c27d4fe885949ea94d1

LABEL maintainer="Fleet Developers"

# Install system dependencies including Go, build-base for make, gcc, g++, and Git if necessary
RUN apk add --no-cache ca-certificates jq curl bash binutils-gold \
    && apk add --no-cache go \
    && apk add --no-cache build-base \
    && apk add --no-cache git

# Install Node.js and Yarn
RUN apk add --no-cache nodejs npm \
    && npm install --global yarn

# Run your make commands
RUN make deps \
    && make generate \
    && make

# Create a non-root user and switch to it
RUN addgroup -S fleet && adduser -S fleet -G fleet
USER fleet

