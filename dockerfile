FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /main tmazleo/api-instagram/cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /main .
COPY run_etl_and_transform.sh .
COPY sql ./sql

RUN chmod +x run_etl_and_transform.sh

CMD ["./run_etl_and_transform.sh"]