FROM golang:1.19-bullseye as BUILDER
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