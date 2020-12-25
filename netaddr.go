package ipam

import (
	"fmt"

	"inet.af/netaddr"
)

func extractPrefix(prefix netaddr.IPPrefix, length uint8) (netaddr.IPPrefix, error) {
	if length <= prefix.Bits {
		return netaddr.IPPrefix{}, fmt.Errorf("length must be greater than prefix.Bits")
	}
	return netaddr.IPPrefix{IP: prefix.IP, Bits: length}, nil
}

func extractPrefixFromSet(set netaddr.IPSet, length uint8) (netaddr.IPPrefix, error) {
	prefixes := set.Prefixes()
	if len(prefixes) == 0 {
		return netaddr.IPPrefix{}, fmt.Errorf("no more child prefixes contained in prefix pool")
	}
	existingPrefixes := make(map[uint8]netaddr.IPPrefix)
	for _, prefix := range prefixes {
		existingPrefixes[prefix.Bits] = prefix
	}
	exactMatch, ok := existingPrefixes[length]
	if ok {
		return exactMatch, nil
	}

	nextBiggerPrefix, ok := existingPrefixes[length-1]
	if !ok {
		if len(prefixes) < 1 {
			return netaddr.IPPrefix{}, fmt.Errorf("no more prefixes left")
		}
		extracted, err := extractPrefix(prefixes[0], length)
		if err != nil {
			return netaddr.IPPrefix{}, err
		}
		return extracted, nil
	}

	extracted, err := extractPrefix(nextBiggerPrefix, length)
	if err != nil {
		return netaddr.IPPrefix{}, err
	}
	return extracted, nil
}
