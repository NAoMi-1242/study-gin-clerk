FROM golang:1.26.7-alpine

WORKDIR /app

RUN apk add --no-cache git

RUN go install github.com/air-verse/air@v1.63.4
RUN go install github.com/google/wire/cmd/wire@v0.7.0
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3

COPY go.mod go.sum ./

RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]