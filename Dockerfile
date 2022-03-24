FROM golang:1.18.0-alpine3.15 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build .


FROM alpine:3.15.2

COPY --from=builder /app/agenda-create /usr/local/bin/agenda-create

RUN apk add git

CMD agenda-create