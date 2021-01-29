# IPv6 Support for metal-stack

Two year ago, when we started to implement metal-stack, we first decided to rely on a external tool to do all the hard IP address management.
For this purpose we used [netbox](https://github.com/netbox-community/netbox) from digitalocean.
To talk to netbox we also implemented our own go library, which was not available back then, because all of metal-stack is written in go.

It turns out that netbox served well for our purpose, creating networks, acquiring ips and releasing them work as expected but was way to slow.
We also found that their api forced us to make a lot of calls for one single purpose.

Because one of our main goals during the development of metal-stack was speed and simplicity we decided to implement our own IPAM (IP Address Management) and [go-ipam](https://github.com/metal-stack/go-ipam) was born.

## go-ipam the beginning

go-ipam should only serve the IPAM requirements from metal-stack and these are:

- managing prefixes (a prefix is a network with a given bitlength or mask)
- managing child prefixes, this is required to get a prefix out of a bigger prefix
- acquire and release of ips from a specified prefix

The interface of go-ipam is simple, we have two structs, IP and Prefix:

```go
type IP struct {
    IP           netaddr.IP
    ParentPrefix string
}

type Prefix struct {
    Cidr         string
    ParentCidr   string
}
```

And the visible interface is:

```go
type Ipamer interface {
    // NewPrefix create a new Prefix from a string notation.
    NewPrefix(cidr string) (*Prefix, error)
    // DeletePrefix delete a Prefix from a string notation.
    // If the Prefix is not found an NotFoundError is returned.
    DeletePrefix(cidr string) (*Prefix, error)
    // AcquireChildPrefix will return a Prefix with a smaller length from the given Prefix.
    AcquireChildPrefix(parentCidr string, length uint8) (*Prefix, error)
    // ReleaseChildPrefix will mark this child Prefix as available again.
    ReleaseChildPrefix(child *Prefix) error
    // PrefixFrom will return a known Prefix.
    PrefixFrom(cidr string) *Prefix
    // AcquireSpecificIP will acquire given IP and mark this IP as used, if already in use, return nil.
    // If specificIP is empty, the next free IP is returned.
    // If there is no free IP an NoIPAvailableError is returned.
    AcquireSpecificIP(prefixCidr, specificIP string) (*IP, error)
    // AcquireIP will return the next unused IP from this Prefix.
    AcquireIP(prefixCidr string) (*IP, error)
    // ReleaseIP will release the given IP for later usage and returns the updated Prefix.
    // If the IP is not found an NotFoundError is returned.
    ReleaseIP(ip *IP) (*Prefix, error)
    // ReleaseIPFromPrefix will release the given IP for later usage.
    // If the Prefix or the IP is not found an NotFoundError is returned.
    ReleaseIPFromPrefix(prefixCidr, ip string) error
}
```

With this we where able to do what we expected, create a prefix and acquire a free ip.
After the ip is not used anymore, we can release the ip, and if no ip is left in the prefix we can release the prefix as well.
The following snippet show you how.

```go
    ipamer := New()
    prefix, err := ipamer.NewPrefix("192.168.0.0/24")
    if err != nil {
        panic(err)
    }
    ip1, err := ipamer.AcquireIP(prefix.Cidr)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Super Prefix  : %s\n", prefix)
    fmt.Printf("Super Prefix IP1 : %s\n", ip1.IP.String())
    fmt.Printf("Super Prefix IP1 Parent : %s\n", ip1.ParentPrefix)
    // Output:
    // Super Prefix  : 192.168.0.0/24
    // Super Prefix IP1 : 192.168.0.1
    // Super Prefix IP1 Parent : 192.168.0.0/24
    _, err = ipamer.ReleaseIP(ip1)
    if err != nil {
        panic(err)
    }
    _, err = ipamer.DeletePrefix(prefix.Cidr)
    if err != nil {
        panic(err)
    }
```

With this implementation we where able to do all IPAM operations in a matter of milliseconds instead of 10s of seconds when using netbox.
So we ripped out netbox from metal-api and used our own go-ipam since then.

## go-ipam open issues

As we wanted to be fast during the implementation of go-ipam, we skipped some ugly parts. We where not able to implement two major things:

- IPv6
- Child prefixes with variable bitlength or mask

Both features require complicated and well tested algorithms for ip addresses. We skipped this effort for the time being.
We also did not had an urgent requirement to support IPv6 because kubernetes was at version 1.13 and IPv6 support was not implemented either.
But we where aware that at a later point in time we have to come back and dive deep into IPv6.

The missing support for variable bitlength for child prefixes was another story. Missing this feature forces us to create several networks "by hand" for some cases.

## the solution

As time went by, we discovered the excellent work from the people behind [tailscale](tailscale.com).
They implemented a alternative network manipulation library for go called [netaddr](github.com/inetaf/netaddr).
In contrast to the `net` package in the go standard library, `inet.af/netaddr` has a more convenient and usable API to manipulate network objects.

The most important are `netaddr.IP.Next()` and `netaddr.IP.Contains`, with them it was a snap to implement `AcquireIP` IPv6 compatible. It took us less than 4 hours to make this work, add additional tests and all passes. Hurray.

By looking into `inet.af/netaddr` we stumbled over some other functions which would enable us to solve the child prefix creation for IPv6 as well. First we implemented the required code into go-ipam, but this would probably also be useful for users of `inet.af/netaddr` as well. We created a pull-request which got finally merged: [#61](https://github.com/inetaf/netaddr/pull/61).

With this all in place we gained a lot, not only full IPv6 support, but also the ability to create a child prefix out of a prefix with a defined bitlength.

Now we could do something like this:

```go
    ipamer := New()
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
    prefix = ipamer.PrefixFrom(prefix.Cidr)
    fmt.Printf("Super Prefix  : %s\n", prefix)
    fmt.Printf("Child Prefix 1: %s\n", cp1)
    fmt.Printf("Child Prefix 2: %s\n", cp2)
    fmt.Printf("Child Prefix 2 IP1: %s\n", ip21.IP)
    // Output:
    // Super Prefix  : 2001:aabb::/48
    // Child Prefix 1: 2001:aabb::/64
    // Child Prefix 2: 2001:aabb:0:1::/72
    // Child Prefix 2 IP1: 2001:aabb:0:1::1
```

As it turns out we got even more. Because we switched all network object manipulation from go standard library `net` to `inet.af/netaddr`, all of our functions got way faster with less memory consumption. Go comes with fantastic benchmarking built in and we will explain howto get most out of this in a later blog post.

## the next steps

go-ipam is the foundation for ip address management in metal-stack and is IPv6 ready now. We are currently in the process of make all dependent parts IPv6 aware as well. This journey will be ready in the next couple of weeks.