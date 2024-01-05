FROM golang:1.21.5-bullseye@sha256:5ebf2823dcead16c6cb1499b1e25ca2aa9b3098ac3ccb7f68cc2f4143535addc as BUILDER
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
