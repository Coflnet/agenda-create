FROM registry.suse.com/bci/golang:1.22 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build .


FROM registry.suse.com/bci/bci-micro:15.6

COPY --from=builder /app/agenda-create /usr/local/bin/agenda-create

CMD agenda-create