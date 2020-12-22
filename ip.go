package ipam

import (
	"inet.af/netaddr"
)

// IP is a single ipaddress.
type IP struct {
	IP           netaddr.IP
	ParentPrefix string
}
