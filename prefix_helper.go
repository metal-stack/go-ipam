package ipam

import (
	"fmt"

	"inet.af/netaddr"
)

func extractPrefix(prefix netaddr.IPPrefix, length uint8) (*netaddr.IPPrefix, error) {
	if length <= prefix.Bits {
		return nil, fmt.Errorf("length must be greater than prefix.Bits")
	}

	subrange := netaddr.IPRange{From: prefix.Range().From, To: prefix.Range().To.Prior()}
	if len(subrange.Prefixes()) < 2 {
		return nil, fmt.Errorf("unable to create child prefix for length:%d", length)
	}
	for _, srp := range subrange.Prefixes() {
		if srp.Bits == length {
			return &srp, nil
		}
	}
	return nil, fmt.Errorf("no prefix with length:%d found in %s", length, prefix)
}

func extractPrefixFromSet(set netaddr.IPSet, length uint8) (*netaddr.IPPrefix, error) {
	if len(set.Prefixes()) == 0 {
		return nil, fmt.Errorf("no more child prefixes contained in prefix pool")
	}
	existingPrefixes := make(map[uint8]netaddr.IPPrefix)
	for _, prefix := range set.Prefixes() {
		existingPrefixes[prefix.Bits] = prefix
	}
	exactMatch, ok := existingPrefixes[length]
	if ok {
		return &exactMatch, nil
	}

	nextBiggerPrefix, ok := existingPrefixes[length-1]
	if !ok {
		extracted, err := extractPrefix(set.Prefixes()[0], length)
		if err != nil {
			return nil, err
		}
		return extracted, nil
	}

	extracted, err := extractPrefix(nextBiggerPrefix, length)
	if err != nil {
		return nil, err
	}
	return extracted, nil
}
