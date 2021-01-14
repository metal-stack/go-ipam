# go-ipam

[![Actions](https://github.com/metal-stack/go-ipam/workflows/build/badge.svg)](https://github.com/metal-stack/go-ipam/actions)
[![GoDoc](https://godoc.org/github.com/metal-stack/go-ipam?status.svg)](https://godoc.org/github.com/metal-stack/go-ipam)
[![Go Report Card](https://goreportcard.com/badge/github.com/metal-stack/go-ipam)](https://goreportcard.com/report/github.com/metal-stack/go-ipam)
[![codecov](https://codecov.io/gh/metal-stack/go-ipam/branch/master/graph/badge.svg)](https://codecov.io/gh/metal-stack/go-ipam)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/metal-stack/go-ipam/blob/master/LICENSE)

go-ipam is a module to handle IP address management. It can operate on networks, prefixes and IPs.

## IP

Most obvious this library is all about IP management. The main purpose is to acquire and release an IP, or a bunch of
IP's from prefixes.

## Prefix

A prefix is a network with IP and mask, typically in the form of *192.168.0.0/24*. To be able to manage IPs you have to create a prefix first.

Example usage:

```go

package main

import (
    "fmt"
    goipam "github.com/metal-stack/go-ipam"
)

func main() {
    // create a ipamer with in memory storage
    ipam := goipam.New()

    prefix, err := ipam.NewPrefix("192.168.0.0/24")
    if err != nil {
        panic(err)
    }

    ip, err := ipam.AcquireIP(prefix.Cidr)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got IP: %s", ip.IP)

    prefix, err = ipam.ReleaseIP(ip)
    if err != nil {
        panic(err)
    }
    fmt.Printf("IP: %s released.", ip.IP)

    // Now a IPv6 Super Prefix with Child Prefixes
    prefix, err := ipamer.NewPrefix("2001:aabb::/48")
    if err != nil {
        panic(err)
    }
    cp1, err := ipamer.AcquireChildPrefix(prefix.Cidr, 64)
    if err != nil {
        panic(err)
    }
    cp2, err := ipamer.AcquireChildPrefix(prefix.Cidr, 72)
    if err != nil {
        panic(err)
    }
    ip21, err := ipamer.AcquireIP(cp2.Cidr)
    if err != nil {
        panic(err)
    }
}
```

## Performance

```bash
BenchmarkNewPrefix/Memory-4             422596       2712 ns/op     1536 B/op    20 allocs/op
BenchmarkNewPrefix/Postgres-4              127    8257821 ns/op     5587 B/op   126 allocs/op
BenchmarkNewPrefix/Cockroach-4              18   78498926 ns/op     5869 B/op   128 allocs/op
BenchmarkAcquireIP/Memory-4             299738       3941 ns/op     2360 B/op    42 allocs/op
BenchmarkAcquireIP/Postgres-4               88   13501419 ns/op    10740 B/op   257 allocs/op
BenchmarkAcquireIP/Cockroach-4              14   79709070 ns/op    11253 B/op   265 allocs/op
BenchmarkAcquireChildPrefix/8/14-4      153535       7995 ns/op     4496 B/op    69 allocs/op
BenchmarkAcquireChildPrefix/8/16-4      151178       8000 ns/op     4496 B/op    69 allocs/op
BenchmarkAcquireChildPrefix/8/20-4      152636       7760 ns/op     4496 B/op    69 allocs/op
BenchmarkAcquireChildPrefix/8/22-4      154134       7793 ns/op     4496 B/op    69 allocs/op
BenchmarkAcquireChildPrefix/8/24-4      153325       7759 ns/op     4496 B/op    69 allocs/op
BenchmarkAcquireChildPrefix/16/18-4     147488       8186 ns/op     4496 B/op    69 allocs/op
BenchmarkAcquireChildPrefix/16/20-4     154622       7802 ns/op     4496 B/op    69 allocs/op
BenchmarkAcquireChildPrefix/16/22-4     154148       8034 ns/op     4496 B/op    69 allocs/op
BenchmarkAcquireChildPrefix/16/24-4     138088       8748 ns/op     4496 B/op    69 allocs/op
BenchmarkAcquireChildPrefix/16/26-4     125978       8104 ns/op     4496 B/op    69 allocs/op
BenchmarkPrefixOverlapping-4           3400266        342 ns/op        0 B/op     0 allocs/op
```