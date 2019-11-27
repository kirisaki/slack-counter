FROM alpine:latest

COPY slack-count /bin/slack-count
RUN chmod u+x /bin/slack-count

ENTRYPOINT ["/bin/slack-counter"]
