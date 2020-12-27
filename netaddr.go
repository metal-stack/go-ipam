package ipam

import (
	"inet.af/netaddr"
)

func extractPrefixFromSet(set netaddr.IPSet, length uint8) (netaddr.IPPrefix, bool) {
	prefixes := set.Prefixes()
	if len(prefixes) == 0 {
		return netaddr.IPPrefix{}, false
	}
	existingPrefixes := make(map[uint8]netaddr.IPPrefix)
	for _, prefix := range prefixes {
		existingPrefixes[prefix.Bits] = prefix
	}
	exactMatch, ok := existingPrefixes[length]
	if ok {
		return exactMatch, true
	}

	nextBiggerPrefix, ok := existingPrefixes[length-1]
	if !ok {
		if len(prefixes) < 1 {
			return netaddr.IPPrefix{}, false
		}
		return netaddr.IPPrefix{IP: prefixes[0].IP, Bits: length}, true
	}

	return netaddr.IPPrefix{IP: nextBiggerPrefix.IP, Bits: length}, true
}
