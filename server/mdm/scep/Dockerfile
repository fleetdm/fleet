FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

COPY ./scepclient-linux-amd64 /usr/bin/scepclient
COPY ./scepserver-linux-amd64 /usr/bin/scepserver

EXPOSE 8080

VOLUME ["/depot"]

ENTRYPOINT ["scepserver"]
