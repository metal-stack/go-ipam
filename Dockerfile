FROM bufbuild/buf:1.6.0 as buf
FROM golang:1.18-alpine as builder

RUN apk add \
    binutils \
    gcc \
    git \
    libc-dev \
    make

WORKDIR /work
COPY --from=buf /usr/local/bin/buf /usr/local/bin/buf
COPY . .
RUN make

FROM alpine:3.16
COPY --from=builder /work/bin/server /
ENTRYPOINT [ "/server" ]