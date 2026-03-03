FROM golang:1.25-alpine

WORKDIR /app

RUN apk add --no-cache curl

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/bin/server ./cmd/server

EXPOSE 8080
ENTRYPOINT ["/app/bin/server"]
