FROM golang:1.21.3-bullseye@sha256:27b021393d0e0dfffc6cd6cca5e7836ac59f5ac98724c5d6b3b0a82199d275c5 as BUILDER
WORKDIR /src

RUN apt update && apt upgrade -y

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-extldflags '-static'"


FROM public.ecr.aws/aws-cli/aws-cli:latest

COPY --from=BUILDER /src/ingress_destroyer /usr/local/bin/ingress_destroyer

RUN chmod +x /usr/local/bin/ingress_destroyer

ENTRYPOINT ["ingress_destroyer"]
