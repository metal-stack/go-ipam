package ipam

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type sql struct {
	db *sqlx.DB
}

// prefixJSON is the on disk representation in a sql database
type prefixJSON struct {
	Prefix
	AvailableChildPrefixes map[string]bool // available child prefixes of this prefix
	// TODO remove this in the next release
	ChildPrefixLength int             // the length of the child prefixes. Legacy to migrate existing prefixes stored in the db to set the IsParent on reads.
	IsParent          bool            // set to true if there are child prefixes
	IPs               map[string]bool // The ips contained in this prefix
	Version           int64           // Version is used for optimistic locking
}

// JSONEncode a prefix into json
func (p *Prefix) JSONEncode() ([]byte, error) {
	pfxj := prefixJSON{
		Prefix: Prefix{
			Cidr:       p.Cidr,
			ParentCidr: p.ParentCidr,
		},
		AvailableChildPrefixes: p.availableChildPrefixes,
		IsParent:               p.isParent,
		IPs:                    p.ips,
		Version:                p.version,
	}
	pj, err := json.Marshal(pfxj)
	if err != nil {
		return nil, fmt.Errorf("unable to encode prefix:%v", err)
	}
	return pj, nil
}

// JSONDecode new Prefix from json given as byte slice
func (p *Prefix) JSONDecode(buf []byte) error {
	var pre prefixJSON
	err := json.Unmarshal(buf, &pre)
	if err != nil {
		return fmt.Errorf("unable to decode prefix:%v", err)
	}
	// Legacy support only on reading from database, convert to isParent.
	// TODO remove this in the next release
	if pre.ChildPrefixLength > 0 {
		pre.IsParent = true
	}
	p.Cidr = pre.Cidr
	p.ParentCidr = pre.ParentCidr
	p.availableChildPrefixes = pre.AvailableChildPrefixes
	p.isParent = pre.IsParent
	p.ips = pre.IPs
	p.version = pre.Version
	return nil
}

func (s *sql) prefixExists(prefix Prefix) (*Prefix, bool) {
	p, err := s.ReadPrefix(prefix.Cidr)
	if err != nil {
		return nil, false
	}
	return &p, true
}

func (s *sql) CreatePrefix(prefix Prefix) (Prefix, error) {
	existingPrefix, exists := s.prefixExists(prefix)
	if exists {
		return *existingPrefix, nil
	}
	prefix.version = int64(0)
	pj, err := prefix.JSONEncode()
	if err != nil {
		return Prefix{}, err
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%v", err)
	}
	tx.MustExec("INSERT INTO prefixes (cidr, prefix) VALUES ($1, $2)", prefix.Cidr, pj)
	return prefix, tx.Commit()
}

func (s *sql) ReadPrefix(prefix string) (Prefix, error) {
	var result []byte
	err := s.db.Get(&result, "SELECT prefix FROM prefixes WHERE cidr=$1", prefix)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read prefix:%v", err)
	}
	var p Prefix
	err = p.JSONDecode(result)
	if err != nil {
		return Prefix{}, err
	}
	return p, nil
}

func (s *sql) ReadAllPrefixes() ([]Prefix, error) {
	var prefixes [][]byte
	err := s.db.Select(&prefixes, "SELECT prefix FROM prefixes")
	if err != nil {
		return nil, fmt.Errorf("unable to read prefixes:%v", err)
	}

	result := []Prefix{}
	for _, v := range prefixes {
		var p Prefix
		err = p.JSONDecode(v)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

// UpdatePrefix tries to update the prefix.
// Returns OptimisticLockError if it does not succeed due to a concurrent update.
func (s *sql) UpdatePrefix(prefix Prefix) (Prefix, error) {
	oldVersion := prefix.version
	prefix.version = oldVersion + 1
	pn, err := prefix.JSONEncode()
	if err != nil {
		return Prefix{}, err
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%v", err)
	}
	result := tx.MustExec("SELECT prefix FROM prefixes WHERE cidr=$1 AND prefix->>'Version'=$2 FOR UPDATE", prefix.Cidr, oldVersion)
	rows, err := result.RowsAffected()
	if err != nil {
		return Prefix{}, err
	}
	if rows == 0 {
		err := tx.Rollback()
		if err != nil {
			return Prefix{}, newOptimisticLockError("select for update did not effect any row, but rollback did not work:" + err.Error())
		}
		return Prefix{}, newOptimisticLockError("select for update did not effect any row")
	}
	result = tx.MustExec("UPDATE prefixes SET prefix=$1 WHERE cidr=$2 AND prefix->>'Version'=$3", pn, prefix.Cidr, oldVersion)
	rows, err = result.RowsAffected()
	if err != nil {
		return Prefix{}, err
	}
	if rows == 0 {
		err := tx.Rollback()
		if err != nil {
			return Prefix{}, newOptimisticLockError("updatePrefix did not effect any row, but rollback did not work:" + err.Error())
		}
		return Prefix{}, newOptimisticLockError("updatePrefix did not effect any row")
	}
	return prefix, tx.Commit()
}

func (s *sql) DeletePrefix(prefix Prefix) (Prefix, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%v", err)
	}
	tx.MustExec("DELETE from prefixes WHERE cidr=$1", prefix.Cidr)
	return prefix, tx.Commit()
}
