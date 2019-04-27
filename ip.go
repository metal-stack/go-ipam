package ipam

import (
	"fmt"
	"math/big"
	"net"
)

// IP is a single ipaddress.
type IP struct {
	IP    net.IP
	IPNet *net.IPNet
}

// AcquireIP will return the next unused IP from this Prefix.
func (i *Ipamer) AcquireIP(prefix Prefix) (*IP, error) {
	prefix.Lock()
	defer prefix.Unlock()
	var acquired *IP
	ipnet, err := prefix.IPNet()
	if err != nil {
		return nil, err
	}
	network, err := prefix.Network()
	if err != nil {
		return nil, err
	}
	for ip := network.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		_, ok := prefix.IPs[ip.String()]
		if !ok {
			acquired = &IP{
				IP:    ip,
				IPNet: ipnet,
			}
			prefix.IPs[ip.String()] = *acquired
			_, err := i.storage.UpdatePrefix(&prefix)
			if err != nil {
				return nil, fmt.Errorf("unable to persist aquired ip:%v", err)
			}
			return acquired, nil
		}
	}
	return nil, nil
}

// ReleaseIP will release the given IP for later usage.
func (i *Ipamer) ReleaseIP(ip IP) error {
	prefix := i.getPrefixOfIP(&ip)
	return i.ReleaseIPFromPrefix(prefix, ip.IP.String())
}

// ReleaseIPFromPrefix will release the given IP for later usage.
func (i *Ipamer) ReleaseIPFromPrefix(prefix *Prefix, ip string) error {
	if prefix == nil {
		return fmt.Errorf("prefix is nil")
	}
	prefix.Lock()
	defer prefix.Unlock()

	_, ok := prefix.IPs[ip]
	if !ok {
		return fmt.Errorf("unable to release ip:%s because it is not allocated in prefix:%s", ip, prefix.Cidr)
	}
	delete(prefix.IPs, ip)
	_, err := i.storage.UpdatePrefix(prefix)
	if err != nil {
		return fmt.Errorf("unable to release ip %v:%v", ip, err)
	}
	return nil
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

func (p *Prefix) broadcast() (*IP, error) {
	ipnet, err := p.IPNet()
	if err != nil {
		return nil, err
	}
	network, err := p.Network()
	if err != nil {
		return nil, err
	}
	mask := ipnet.Mask
	n := IP{IP: network}
	m := IP{IP: net.IP(mask)}

	broadcast := n.or(m.not())
	return &broadcast, nil
}

func (i *IP) lshift(bits uint8) IP {
	var result []byte
	for _, part := range i.IP {
		result = append(result, part<<bits)
	}

	return IP{
		IP: result,
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
