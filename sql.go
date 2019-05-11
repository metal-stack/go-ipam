package ipam

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type sql struct {
	db *sqlx.DB
}

// https://stackoverflow.com/questions/28035784/golang-marshal-unmarshal-json-with-both-exported-and-un-exported-fields?rq=1
type prefixAlias Prefix

type prefixJSON struct {
	*prefixAlias
	AvailableChildPrefixes map[string]bool // available child prefixes of this prefix
	ChildPrefixLength      int             // the length of the child prefixes
	IPs                    map[string]bool // The ips contained in this prefix
}

// MarshalJSON marshals a Prefix. (struct to JSON)
func (p *Prefix) MarshalJSON() ([]byte, error) {
	return json.Marshal(&prefixJSON{
		prefixAlias: (*prefixAlias)(p),
		// Unexported fields are listed here:
		AvailableChildPrefixes: p.availableChildPrefixes,
		ChildPrefixLength:      p.childPrefixLength,
		IPs:                    p.ips,
	})
}

// UnmarshalJSON unmarshals a Prefix. (JSON to struct)
func (p *Prefix) UnmarshalJSON(data []byte) error {
	temp := &prefixJSON{
		prefixAlias: (*prefixAlias)(p),
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy the exported fields:
	*p = (Prefix)(*(temp).prefixAlias)
	// Each unexported field must be copied and/or converted individually:
	p.availableChildPrefixes = temp.AvailableChildPrefixes
	p.childPrefixLength = temp.childPrefixLength
	p.ips = temp.IPs
	return nil
}

func (s *sql) prefixExists(prefix *Prefix) (*Prefix, bool) {
	p, err := s.ReadPrefix(prefix.Cidr)
	if err != nil {
		return nil, false
	}
	if p == nil {
		return nil, false
	}
	return p, true
}

func (s *sql) CreatePrefix(prefix *Prefix) (*Prefix, error) {
	existingPrefix, exists := s.prefixExists(prefix)
	if exists {
		return existingPrefix, nil
	}
	tx := s.db.MustBegin()
	pj, err := json.Marshal(prefix)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal prefix:%v", err)
	}
	tx.MustExec("INSERT INTO prefixes (cidr, prefix) VALUES ($1, $2)", prefix.Cidr, pj)
	return prefix, tx.Commit()
}

func (s *sql) ReadPrefix(prefix string) (*Prefix, error) {
	var result []byte
	err := s.db.Get(&result, "SELECT prefix FROM prefixes WHERE cidr=$1", prefix)
	if err != nil {
		return nil, fmt.Errorf("unable to read prefix:%v", err)
	}
	var pre Prefix
	err = json.Unmarshal(result, &pre)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal prefix:%v", err)
	}

	return &pre, nil
}

func (s *sql) ReadAllPrefixes() ([]*Prefix, error) {
	var prefixes [][]byte
	err := s.db.Select(&prefixes, "SELECT prefix FROM prefixes")
	if err != nil {
		return nil, fmt.Errorf("unable to read prefixes:%v", err)
	}

	result := []*Prefix{}
	for _, v := range prefixes {
		var pre Prefix
		err = json.Unmarshal(v, &pre)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal prefix:%v", err)
		}
		result = append(result, &pre)
	}
	return result, nil
}

func (s *sql) UpdatePrefix(prefix *Prefix) (*Prefix, error) {
	tx := s.db.MustBegin()
	pn, err := json.Marshal(prefix)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal prefix:%v", err)
	}
	tx.MustExec("UPDATE prefixes SET prefix=$1 WHERE cidr=$2", pn, prefix.Cidr)
	return prefix, tx.Commit()
}

func (s *sql) DeletePrefix(prefix *Prefix) (*Prefix, error) {
	tx := s.db.MustBegin()
	tx.MustExec("DELETE from prefixes WHERE cidr=$1", prefix.Cidr)
	return prefix, tx.Commit()
}
