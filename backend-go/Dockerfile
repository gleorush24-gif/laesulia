FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o laesulia-api ./cmd/server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
RUN mkdir -p /app/uploads
WORKDIR /app
COPY --from=builder /app/laesulia-api .
EXPOSE 8080
CMD ["./laesulia-api"]
