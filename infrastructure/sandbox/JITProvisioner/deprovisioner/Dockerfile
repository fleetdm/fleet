FROM golang:1.21.7-alpine@sha256:0ff68fa7b2177e8d68b4555621c2321c804bcff839fd512c2681de49026573b7 AS builder
RUN apk update && apk add --no-cache git curl openssl unzip
WORKDIR /build
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-extldflags '-static'"
RUN curl https://releases.hashicorp.com/terraform/1.1.8/terraform_1.1.8_linux_amd64.zip > terraform.zip
RUN unzip terraform.zip
RUN rm terraform.zip
RUN chmod 644 $(find . -type f)
RUN chmod 755 $(find . -type d)
RUN chmod 655 lambda terraform

#FROM scratch
#COPY --from=builder /build/lambda /build/terraform /
#COPY --from=builder /build/deploy_terraform /deploy_terraform
#COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/build/lambda"]
