FROM debian:bookworm-slim@sha256:ccb33c3ac5b02588fc1d9e4fc09b952e433d0c54d8618d0ee1afadf1f3cf2455

WORKDIR /usr/src/app

COPY test_tuf file-server ./

CMD ["./file-server", "8081", "repository"]
