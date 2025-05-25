FROM golang:1.21.6-alpine3.19 AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/api cmd/api/main.go

# ----------------------------

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/bin/api .
COPY --from=builder /app/.env .

RUN ls -lah

ENTRYPOINT [ "./api" ]
