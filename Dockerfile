FROM golang:1.21-alpine as builder

RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go build -o crtsher ./cmd/...
RUN chmod +x ./crtsher
ENTRYPOINT ["/app/crtsher"]
