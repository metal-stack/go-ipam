FROM alpine:3.20 as health-downloader
ENV GRPC_HEALTH_PROBE_VERSION=v0.4.26 \
    GRPC_HEALTH_PROBE_URL=https://github.com/grpc-ecosystem/grpc-health-probe/releases/download
RUN apk -U add curl \
 && curl -fLso /bin/grpc_health_probe \
    ${GRPC_HEALTH_PROBE_URL}/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 \
 && chmod +x /bin/grpc_health_probe
 
FROM golang:1.22-alpine as builder
RUN apk add \
    binutils \
    gcc \
    git \
    libc-dev \
    make

WORKDIR /work
COPY . .
RUN make server client

FROM alpine:3.20
COPY --from=health-downloader /bin/grpc_health_probe /bin/grpc_health_probe
COPY --from=builder /work/bin/* /
ENTRYPOINT [ "/server" ]