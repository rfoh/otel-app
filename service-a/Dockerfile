FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/service-a ./...

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/service-a /usr/local/bin/service-a

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/service-a"]
