FROM golang:1.19.8-alpine@sha256:86d32cc0dfc04757fd8aeebb86308e6d1e3de60c73cb59e0f99c7b2ef77416b6 AS builder
WORKDIR /build
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-extldflags '-static'"

#FROM scratch
#COPY --from=builder /build/lambda /build/terraform /
#COPY --from=builder /build/deploy_terraform /deploy_terraform
#COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/build/lambda"]
