FROM alpine:latest

COPY slack-counter /bin/slack-counter
RUN chmod u+x /bin/slack-counter

ENTRYPOINT ["/bin/slack-counter"]
