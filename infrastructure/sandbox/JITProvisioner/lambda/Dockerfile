FROM golang:1.21.3-alpine@sha256:27c76dcf886c5024320f4fa8ceb57d907494a3bb3d477d0aa7ac8385acd871ea AS builder
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
