FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o /app/main ./

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

COPY --from=builder /app/static ./static
COPY --from=builder /app/templates ./templates

EXPOSE 4040

CMD ["./main"]
