FROM debian:bookworm-slim

WORKDIR /usr/src/app

COPY test_tuf file-server .

CMD ["./file-server", "8081", "test_tuf/repository"] 
