FROM golang:1.26-trixie AS builder

WORKDIR /work
COPY . .
RUN make server client

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /work/bin/* /
ENTRYPOINT [ "/server" ]