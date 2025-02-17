FROM golang:1.24.0-alpine3.21 as builder

WORKDIR /src
COPY . .
RUN go build -o nats_receive .

FROM scratch

COPY --from=builder /src/nats_receive /app/nats_receive
CMD ["/app/nats_receive"]
