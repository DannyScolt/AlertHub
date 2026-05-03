FROM golang:1.25-alpine

WORKDIR /app

RUN apk add --no-cache git curl
RUN go install github.com/air-verse/air@v1.65.1

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
