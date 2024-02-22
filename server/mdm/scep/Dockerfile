FROM alpine:3

COPY ./scepclient-linux-amd64 /usr/bin/scepclient
COPY ./scepserver-linux-amd64 /usr/bin/scepserver

EXPOSE 8080

VOLUME ["/depot"]

ENTRYPOINT ["scepserver"]
