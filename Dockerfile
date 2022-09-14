FROM bufbuild/buf:1.7.0 as buf
FROM golang:1.19-alpine as builder

RUN apk add \
    binutils \
    gcc \
    git \
    libc-dev \
    make

WORKDIR /work
COPY --from=buf /usr/local/bin/buf /usr/local/bin/buf
COPY . .
RUN make server client

FROM alpine:3.16
COPY --from=builder /work/bin/* /
ENTRYPOINT [ "/server" ]