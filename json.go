package ipam

import (
	"encoding/json"
	"fmt"
)

// PrefixJSON is the serializable representation of a Prefix.
// It is exported so that external Storage implementations can persist and
// restore prefixes without having to know about the unexported fields of
// Prefix.
type PrefixJSON struct {
	Prefix
	Namespace              string          `json:"Namespace"`
	AvailableChildPrefixes map[string]bool `json:"AvailableChildPrefixes"` // available child prefixes of this prefix
	// TODO remove this in the next release
	ChildPrefixLength int             `json:"ChildPrefixLength"` // the length of the child prefixes. Legacy to migrate existing prefixes stored in the db to set the IsParent on reads.
	IsParent          bool            `json:"IsParent"`          // set to true if there are child prefixes
	IPs               map[string]bool `json:"IPs"`               // The ips contained in this prefix
	Version           int64           `json:"Version"`           // Version is used for optimistic locking
}

// ToPrefix converts the serializable representation back into a Prefix.
func (p PrefixJSON) ToPrefix() Prefix {
	// Legacy support only on reading from database, convert to isParent.
	// TODO remove this in the next release
	if p.ChildPrefixLength > 0 {
		p.IsParent = true
	}
	return Prefix{
		Cidr:                   p.Cidr,
		ParentCidr:             p.ParentCidr,
		availableChildPrefixes: p.AvailableChildPrefixes,
		childPrefixLength:      p.ChildPrefixLength,
		isParent:               p.IsParent,
		ips:                    p.IPs,
		version:                p.Version,
	}
}

// ToPrefixJSON converts this Prefix into its serializable representation.
func (p *Prefix) ToPrefixJSON() PrefixJSON {
	return PrefixJSON{
		Cidr:                   p.Cidr,
		ParentCidr:             p.ParentCidr,
		AvailableChildPrefixes: p.availableChildPrefixes,
		IsParent:               p.isParent,
		// TODO remove this in the next release
		ChildPrefixLength: p.childPrefixLength,
		IPs:               p.ips,
		Version:           p.version,
	}
}

// ToJSON marshals this Prefix into its JSON representation.
func (p *Prefix) ToJSON() ([]byte, error) {
	pj, err := json.Marshal(p.ToPrefixJSON()) // nolint:musttag
	if err != nil {
		return nil, fmt.Errorf("unable to marshal prefix:%w", err)
	}
	return pj, nil
}

// ToJSON marshals these Prefixes into their JSON representation.
func (ps Prefixes) ToJSON() ([]byte, error) {
	var pfxjs []PrefixJSON
	for _, p := range ps {
		pfxjs = append(pfxjs, p.ToPrefixJSON())
	}
	pj, err := json.Marshal(pfxjs)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal prefixes:%w", err)
	}
	return pj, nil
}

// FromJSON unmarshals a Prefix from its JSON representation.
func FromJSON(js []byte) (Prefix, error) {
	var pre PrefixJSON
	err := json.Unmarshal(js, &pre) // nolint:musttag
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to unmarshal prefix:%w", err)
	}
	return pre.ToPrefix(), nil
}

// FromJSONs unmarshals a list of Prefixes from their JSON representation.
func FromJSONs(js []byte) (Prefixes, error) {
	var pres []PrefixJSON
	err := json.Unmarshal(js, &pres)
	if err != nil {
		return Prefixes{}, fmt.Errorf("unable to unmarshal prefixes:%w", err)
	}
	var pfxs Prefixes
	for _, pj := range pres {
		pfxs = append(pfxs, pj.ToPrefix())
	}
	return pfxs, nil
}
