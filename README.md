# go-ipam

[![Build Status](https://travis-ci.org/metal-pod/go-ipam.svg?branch=master)](https://travis-ci.org/metal-pod/go-ipam)
[![codecov](https://codecov.io/gh/metal-pod/go-ipam/branch/master/graph/badge.svg)](https://codecov.io/gh/metal-pod/go-ipam)

go-ipam is a module to handle IPAddress management. It can operate on Networks, Prefixes and IPs.

## IP

Most obvious this library is all about ip management. the main purpose is to acquire and release an ip, or a bunch of
ip's from prefixes.

## Prefix

A prefix is a network with ip and mask, typicaly in the form of *192.168.0.0/24*. To be able to manage IPs you have to create a prefix first.

example usage:

```go

package main

import (
    "fmt"
    goipam "github.com/metal-pod/go-ipam"
)


func main() {
    // create a ipamer with in memory storage
    ipam := goipam.New()

    prefix, err := ipam.NewPrefix("192.168.0.0/24")
    if err != nil {
        panic(err)
    }

    ip, err := ipam.AcquireIP(prefix)
    if err != nil {
        panic(err)
    }
    fmt.Printf("got IP: %s", ip.IP)

    err = ipam.ReleaseIP(ip)
    if err != nil {
        panic(err)
    }
    fmt.Printf("IP: %s released.", ip.IP)
}
```
