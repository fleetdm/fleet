FROM debian:stable-slim@sha256:0f116858482fd8222b4f7e9b4cdc9a054051e67fbb8a57bc22651f0d56b45ad8 AS builder

RUN apt-get update
RUN apt-get install -y build-essential autoconf libxml2-dev libssl-dev zlib1g-dev curl git

# Build bomutils
RUN git clone -b master \
    --depth=1 --no-tags --progress \
    --no-recurse-submodules https://github.com/hogliux/bomutils.git && \
    cd bomutils && make && make install

# Install xar
RUN curl -L https://github.com/mackyle/xar/archive/refs/tags/xar-1.6.1.tar.gz > xar.tar.gz && \
    echo "5e7d50dab73f5cb1713b49fa67c455c2a0dd2b0a7770cbc81b675e21f6210e25  xar.tar.gz" | sha256sum --check && \
    tar -xzf xar.tar.gz 
# Note this needs patching due to newer version of OpenSSL
# See https://github.com/mackyle/xar/pull/23
COPY patch.txt .
RUN cd xar-xar-1.6.1/xar && patch < ../../patch.txt && autoconf && ./configure && make && make install


FROM debian:stable-slim@sha256:0f116858482fd8222b4f7e9b4cdc9a054051e67fbb8a57bc22651f0d56b45ad8

RUN apt-get update && apt-get install -y --no-install-recommends libxml2  && rm -rf /var/lib/apt/lists/*
COPY --from=builder /usr/bin /usr/bin/
COPY --from=builder /usr/local/bin /usr/local/bin/
COPY --from=builder /usr/local/lib /usr/local/lib/
