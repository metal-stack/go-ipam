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
BenchmarkNewPrefixMemory-4                484093              2436 ns/op            1536 B/op         20 allocs/op
BenchmarkNewPrefixPostgres-4                 116          10917631 ns/op            5869 B/op        128 allocs/op
BenchmarkNewPrefixCockroach-4                 15          79193457 ns/op            7325 B/op        145 allocs/op
BenchmarkAcquireIPMemory-4                306850              4429 ns/op            2360 B/op         42 allocs/op
BenchmarkAcquireIPPostgres-4                  81          14142993 ns/op           11007 B/op        260 allocs/op
BenchmarkAcquireIPCockroach-4                 13          87037384 ns/op           12970 B/op        286 allocs/op
BenchmarkAcquireChildPrefix1-4            123308              8749 ns/op            4496 B/op         69 allocs/op
BenchmarkAcquireChildPrefix2-4            154058              7707 ns/op            4496 B/op         69 allocs/op
BenchmarkAcquireChildPrefix3-4            156578              8435 ns/op            4496 B/op         69 allocs/op
BenchmarkAcquireChildPrefix4-4            141354              8225 ns/op            4496 B/op         69 allocs/op
BenchmarkAcquireChildPrefix5-4            156516              8087 ns/op            4496 B/op         69 allocs/op
BenchmarkAcquireChildPrefix6-4            138122              8020 ns/op            4496 B/op         69 allocs/op
BenchmarkAcquireChildPrefix7-4            155088              8748 ns/op            4496 B/op         69 allocs/op
BenchmarkAcquireChildPrefix8-4            154384              9105 ns/op            4496 B/op         69 allocs/op
BenchmarkAcquireChildPrefix9-4            141003              8469 ns/op            4496 B/op         69 allocs/op
BenchmarkAcquireChildPrefix10-4           125125              8292 ns/op            4496 B/op         69 allocs/op
BenchmarkPrefixOverlapping-4             3290104               359 ns/op               0 B/op          0 allocs/op
```