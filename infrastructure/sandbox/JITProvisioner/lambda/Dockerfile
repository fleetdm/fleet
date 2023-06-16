FROM golang:1.19.10-alpine@sha256:6b39e22d325eb4c57faa0931ca199c26874b3cd8908caed0761d8fc3356e3750 AS builder
WORKDIR /build
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-extldflags '-static'"
RUN chmod +r standard-query-library.yml

#FROM scratch
#COPY --from=builder /build/lambda /build/terraform /
#COPY --from=builder /build/deploy_terraform /deploy_terraform
#COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/build/lambda"]
