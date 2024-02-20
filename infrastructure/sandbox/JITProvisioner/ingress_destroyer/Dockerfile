FROM golang:1.21.7-bullseye@sha256:447afe790df28e0bc19d782a9f776a105ce3b8417cdd21f33affc4ed6d38f9d5 as BUILDER
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
