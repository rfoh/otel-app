FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/service-b ./...

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/service-b /usr/local/bin/service-b

EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/service-b"]
