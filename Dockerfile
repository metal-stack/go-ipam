FROM golang:1.21-alpine as builder
RUN apk add \
    binutils \
    gcc \
    git \
    libc-dev \
    make

WORKDIR /work
COPY . .
RUN make server client

FROM alpine:3.18
COPY --from=builder /work/bin/* /
ENTRYPOINT [ "/server" ]