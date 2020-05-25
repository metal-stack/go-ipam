package ipam

import (
	"fmt"
	"math/big"
	"net"
)

// IP is a single ipaddress.
type IP struct {
	IP           net.IP
	ParentPrefix string
	UUID         string
}

func (i *IP) or(ip IP) IP {
	var result []byte
	for index, part := range i.IP {
		result = append(result, ip.IP[index]|part)
	}

	return IP{
		IP: result,
	}
}

func (i *IP) not() IP {
	var result []byte
	for _, part := range i.IP {
		result = append(result, ^part)
	}
	return IP{
		IP: result,
	}
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func ipToInt(ip net.IP) (*big.Int, int) {
	val := &big.Int{}
	val.SetBytes([]byte(ip))
	if len(ip) == net.IPv4len {
		return val, 32
	} else if len(ip) == net.IPv6len {
		return val, 128
	}
	return nil, 0
}

func intToIP(ipInt *big.Int, bits int) net.IP {
	ipBytes := ipInt.Bytes()
	ret := make([]byte, bits/8)
	// Pack our IP bytes into the end of the return array,
	// since big.Int.Bytes() removes front zero padding.
	for i := 1; i <= len(ipBytes); i++ {
		ret[len(ret)-i] = ipBytes[len(ipBytes)-i]
	}
	return net.IP(ret)
}

func insertNumIntoIP(ip net.IP, num int, prefixLen int) (*net.IP, error) {
	ipInt, totalBits := ipToInt(ip)
	if ipInt == nil {
		return nil, fmt.Errorf("unable to convert ip %s to int", ip)
	}
	bigNum := big.NewInt(int64(num))
	bigNum.Lsh(bigNum, uint(totalBits-prefixLen))
	ipInt.Or(ipInt, bigNum)
	result := intToIP(ipInt, totalBits)
	return &result, nil
}
