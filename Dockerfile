FROM alpine:latest
Add manager /manager
WORKDIR /
ENTRYPOINT ["/manager"]