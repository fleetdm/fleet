FROM alpine:3.4
MAINTAINER Kolide Developers <engineering@kolide.co>

COPY ./build/kolide /kolide

CMD ["/kolide", "serve"]
