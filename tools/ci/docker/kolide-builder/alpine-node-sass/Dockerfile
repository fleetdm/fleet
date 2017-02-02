FROM mhart/alpine-node:latest
ENV NODE_SASS_VERSION=v3.10.0

RUN apk --update add \
    git \
    build-base \
    perl \
    python

COPY build.sh build.sh
CMD ["/build.sh"]
