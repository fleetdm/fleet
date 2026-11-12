FROM debian:trixie-slim@sha256:f6e2cfac5cf956ea044b4bd75e6397b4372ad88fe00908045e9a0d21712ae3ba

WORKDIR /usr/src/app

COPY test_tuf file-server ./

CMD ["./file-server", "8081", "repository"]
