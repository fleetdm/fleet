FROM golang:1.18-alpine AS builder
RUN apk update && apk add --no-cache git
WORKDIR /build
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-extldflags '-static'"

FROM scratch
COPY --from=builder /build/lambda /lambda
ENTRYPOINT ["/lambda"]
