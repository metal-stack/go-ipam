# go-ipam

[![Actions](https://github.com/metal-stack/go-ipam/actions/workflows/docker.yml/badge.svg?branch=master)](https://github.com/metal-stack/go-ipam/actions)
[![GoDoc](https://godoc.org/github.com/metal-stack/go-ipam?status.svg)](https://godoc.org/github.com/metal-stack/go-ipam)
[![Go Report Card](https://goreportcard.com/badge/github.com/metal-stack/go-ipam)](https://goreportcard.com/report/github.com/metal-stack/go-ipam)
[![codecov](https://codecov.io/gh/metal-stack/go-ipam/branch/master/graph/badge.svg)](https://codecov.io/gh/metal-stack/go-ipam)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/metal-stack/go-ipam/blob/master/LICENSE)

go-ipam is a module to handle IP address management. It can operate on networks, prefixes and IPs.

It also comes as a ready to go microservice which offers a grpc api.

## IP

Most obvious this library is all about IP management. The main purpose is to acquire and release an IP, or a bunch of
IP's from prefixes.

## Prefix

A prefix is a network with IP and mask, typically in the form of *192.168.0.0/24*. To be able to manage IPs you have to create a prefix first.

Library Example usage:

```go

package main

import (
    "context"
    "fmt"
    "time"

    goipam "github.com/metal-stack/go-ipam"
)

func main() {
    // The background context
    bgCtx := context.Background()

    // Create a ipamer with in memory storage
    ipam := goipam.New(bgCtx)

    // Optionally, we can pass around a context for a given namespace
    namespace := "tenant-a"
    err := ipam.CreateNamespace(bgCtx, namespace)
    if err != nil {
        panic(err)
    }
    ctx := goipam.NewContextWithNamespace(bgCtx, namespace)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // Create a prefix to manage some IPs
    prefix, err := ipam.NewPrefix(ctx, "192.168.0.0/24")
    if err != nil {
        panic(err)
    }

    // Acquire and release an IP with this prefix
    ip, err := ipam.AcquireIP(ctx, prefix.Cidr)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got IP: %s\n", ip.IP)

    prefix, err = ipam.ReleaseIP(ctx, ip)
    if err != nil {
        panic(err)
    }
    fmt.Printf("IP: %s released.\n", ip.IP)

    // Now a IPv6 Super Prefix with Child Prefixes
    prefix, err = ipam.NewPrefix(ctx, "2001:aabb::/48")
    if err != nil {
        panic(err)
    }

    cp1, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 64)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got Prefix: %s\n", cp1)

    cp2, err := ipam.AcquireChildPrefix(ctx, prefix.Cidr, 72)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got Prefix: %s\n", cp2)
    ip21, err := ipam.AcquireIP(ctx, cp2.Cidr)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got IP: %s\n", ip21.IP)
}
```

## GRPC Service

First start the go-ipam container with the database backend of your choice already up and running. For example if you have a postgres database for storing the ipam data, you could run the grpc service like so:

```bash
docker run -it --rm ghcr.io/metal-stack/go-ipam postgres
```

From a client perspective you can now talk to this service via grpc.

GRPC Example usage:

```go
package main

import (
    "http"

    "github.com/bufbuild/connect-go"
    v1 "github.com/metal-stack/go-ipam/api/v1"
    "github.com/metal-stack/go-ipam/api/v1/apiv1connect"
)
func main() {

    c := apiv1connect.NewIpamServiceClient(
            http.DefaultClient,
            "http://localhost:9090",
            connect.WithGRPC(),
    )

    bgCtx := context.Background()

    // Optional with Namespace
    ctx := goipam.NewContextWithNamespace(bgCtx, "tenant-a")

    result, err := c.CreatePrefix(ctx, connect.NewRequest(&v1.CreatePrefixRequest{Cidr: "192.168.0.0/16",}))
    if err != nil {
        panic(err)
    }
    fmt.Println("Prefix:%q created", result.Msg.GetPrefix().GetCidr())
}
```

## GRPC client

There is also a `cli` provided in the container which can be used to make calls to the grpc endpoint manually:

```bash
docker run -it --rm --entrypoint /cli ghcr.io/metal-stack/go-ipam
```

## Metrics

```bash
http://localhost:2112/metrics
```

## pprof

```bash
go tool pprof -http :8080 localhost:2113/debug/pprof/heap
go tool pprof -http :8080 localhost:2113/debug/pprof/goroutine
```

## Docker Compose example

Ensure you have docker with compose support installed. Then execute the following command:

```bash
docker compose up -d

# check if up and running
docker compose ps

NAME                 IMAGE             COMMAND                  SERVICE    CREATED          STATUS                    PORTS
go-ipam-ipam-1       go-ipam           "/server postgres"       ipam       14 seconds ago   Up 13 seconds (healthy)   0.0.0.0:9090->9090/tcp, :::9090->9090/tcp
go-ipam-postgres-1   postgres:alpine   "docker-entrypoint.s…"   postgres   8 minutes ago    Up 13 seconds             5432/tcp


# Then execute the cli to create prefixes and acquire ips

docker compose exec ipam /cli prefix create --cidr 192.168.0.0/16
prefix:"192.168.0.0/16" created

docker compose exec ipam /cli ip acquire --prefix  192.168.0.0/16
ip:"192.168.0.1" acquired

# Queries can also made against the Rest api like so:

curl -v -X POST -d '{}' -H 'Content-Type: application/json' localhost:9090/api.v1.IpamService/ListPrefixes
```

## Supported Databases & Performance

| Database    | Acquire Child Prefix |  Acquire IP |  New Prefix | Prefix Overlap | Production-Ready | Geo-Redundant |
|:------------|---------------------:|------------:|------------:|---------------:|:-----------------|:--------------|
| In-Memory   |          106,861/sec | 196,687/sec | 330,578/sec |        248/sec | N                | N             |
| File        |                      |             |             |                | N                | N             |
| KeyDB       |              777/sec |     975/sec |   2,271/sec |                | Y                | Y             |
| Redis       |              773/sec |     958/sec |   2,349/sec |                | Y                | N             |
| MongoDB     |              415/sec |     682/sec |     772/sec |                | Y                | Y             |
| Etcd        |              258/sec |     368/sec |     533/sec |                | Y                | N             |
| Postgres    |              203/sec |     331/sec |     472/sec |                | Y                | N             |
| CockroachDB |              170/sec |     300/sec |     470/sec |                | Y                | Y             |

The benchmarks above were performed using:

* cpu: Intel(R) Xeon(R) Platinum 8370C CPU @ 2.80GHz
* postgres:17-alpine
* cockroach:v24.1.0
* redis:7.4-alpine
* keydb:alpine_x86_64_v6.3.1
* etcd:v3.5.15
* mongodb:7

### Database Version Compatibility

| Database    | Details                                                                                                                   |
|-------------|---------------------------------------------------------------------------------------------------------------------------|
| KeyDB       |                                                                                                                           |
| Redis       |                                                                                                                           |
| MongoDB     | [mongodb-go compatibility](https://www.mongodb.com/docs/drivers/go/current/compatibility/#std-label-golang-compatibility) |
| Etcd        |                                                                                                                           |
| Postgres    |                                                                                                                           |
| CockroachDB |                                                                                                                           |

## Testing individual Backends

It is possible to test a individual backend only to speed up development roundtrip.

`backend` can be one of `Memory`, `Postgres`, `Cockroach`, `Etcd`, `Redis`, and `MongoDB`.

```bash
BACKEND=backend make test
```
