# Use a base image that includes necessary tools
FROM alpine:3.18.2@sha256:82d1e9d7ed48a7523bdebc18cf6290bdb97b82302a8a9c27d4fe885949ea94d1

LABEL maintainer="Fleet Developers"

# Install system dependencies including Go, build-base for make, gcc, g++, and Git if necessary
RUN apk add --no-cache ca-certificates jq curl bash binutils-gold \
    && apk add --no-cache go \
    && apk add --no-cache build-base \
    && apk add --no-cache git

# Optionally, install Node.js and Yarn if your application requires Node.js dependencies
RUN apk add --no-cache nodejs npm \
    && npm install --global yarn

# Set the working directory
WORKDIR /app

# Copy the entire project
COPY . /app

# Run your make commands
RUN make deps \
    && make generate \
    && make

# Set up the executable path (Assuming the make file creates the executable in the current directory or a bin directory)
# You'll need to adjust the path to where the actual executable is being created
COPY ./build/binary-bundle/linux/fleet /usr/bin/fleet

# Create a non-root user and switch to it
RUN addgroup -S fleet && adduser -S fleet -G fleet
USER fleet

# Command to run your application
CMD ["/usr/bin/fleet", "serve"]
