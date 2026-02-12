FROM alpine:3.23 AS health-downloader
ENV GRPC_HEALTH_PROBE_VERSION=v0.4.42 \
    GRPC_HEALTH_PROBE_URL=https://github.com/grpc-ecosystem/grpc-health-probe/releases/download
RUN apk add curl \
 && curl -fLso /bin/grpc_health_probe \
    ${GRPC_HEALTH_PROBE_URL}/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 \
 && chmod +x /bin/grpc_health_probe
 
FROM golang:1.26-alpine AS builder
RUN apk add \
    binutils \
    gcc \
    git \
    libc-dev \
    make

WORKDIR /work
COPY . .
RUN make server client

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=health-downloader /bin/grpc_health_probe /bin/grpc_health_probe
COPY --from=builder /work/bin/* /
ENTRYPOINT [ "/server" ]