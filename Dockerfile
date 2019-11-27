FROM golang:1.13.4-alpine3.10 AS builder

WORKDIR /go/src/slack-counter

COPY . .
RUN mkdir -p /build
RUN go build -o=/build/slack-counter

FROM alpine:3.10.2

COPY --from=builder /build/slack-counter /build/slack-counter
RUN chmod u+x /build/slack-counter

ENTRYPOINT ["/build/slack-counter"]
