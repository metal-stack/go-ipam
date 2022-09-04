package ipam

import (
	"net/netip"
)

// IP is a single ipaddress.
type IP struct {
	IP           netip.Addr
	ParentPrefix string
}
