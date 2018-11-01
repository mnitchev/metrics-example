FROM alpine:latest

COPY metrics /

ENTRYPOINT [ "/metrics" ]
