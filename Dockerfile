FROM golang:1.13.4-alpine3.10 AS server
WORKDIR /go/src/slack-counter
COPY . .
RUN mkdir -p /build
RUN go build -o=/build/slack-counter

FROM node:10.17.0-alpine3.10 AS client
RUN mkdir /client
WORKDIR /go/src/slack-counter
COPY . .
RUN rm -rf node_modules
RUN npm i
RUN npx webpack --production
RUN mv ./dist/* /client

FROM alpine:3.10.2
COPY --from=server /build/slack-counter /build/slack-counter
RUN chmod u+x /build/slack-counter
COPY --from=client /client /client
ENTRYPOINT ["/build/slack-counter"]
