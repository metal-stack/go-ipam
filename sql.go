package ipam

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type sql struct {
	db *sqlx.DB
}

type prefixJSON struct {
	Prefix
	AvailableChildPrefixes map[string]bool // available child prefixes of this prefix
	// TODO remove this in the next release
	ChildPrefixLength int             // the length of the child prefixes. Legacy to migrate existing prefixes stored in the db to set the IsParent on reads.
	IsParent          bool            // set to true if there are child prefixes
	IPs               map[string]bool // The ips contained in this prefix
	Version           int64           // Version is used for optimistic locking
}

func (p prefixJSON) toPrefix() Prefix {
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

func (p Prefix) toPrefixJSON() prefixJSON {
	return prefixJSON{
		Prefix: Prefix{
			Cidr:       p.Cidr,
			ParentCidr: p.ParentCidr,
		},
		AvailableChildPrefixes: p.availableChildPrefixes,
		IsParent:               p.isParent,
		// TODO remove this in the next release
		ChildPrefixLength: p.childPrefixLength,
		IPs:               p.ips,
		Version:           p.version,
	}
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
	pj, err := json.Marshal(prefix.toPrefixJSON())
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to marshal prefix:%w", err)
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%w", err)
	}
	_, err = tx.Exec("INSERT INTO prefixes (cidr, prefix) VALUES ($1, $2)", prefix.Cidr, pj)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to insert prefix:%w", err)
	}
	return prefix, tx.Commit()
}

func (s *sql) ReadPrefix(prefix string) (Prefix, error) {
	var result []byte
	err := s.db.Get(&result, "SELECT prefix FROM prefixes WHERE cidr=$1", prefix)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read prefix:%w", err)
	}
	var pre prefixJSON
	err = json.Unmarshal(result, &pre)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to unmarshal prefix:%w", err)
	}
	return pre.toPrefix(), nil
}

func (s *sql) ReadAllPrefixes() ([]Prefix, error) {
	var prefixes [][]byte
	err := s.db.Select(&prefixes, "SELECT prefix FROM prefixes")
	if err != nil {
		return nil, fmt.Errorf("unable to read prefixes:%w", err)
	}

	result := []Prefix{}
	for _, v := range prefixes {
		var pre prefixJSON
		err = json.Unmarshal(v, &pre)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal prefix:%w", err)
		}
		result = append(result, pre.toPrefix())
	}
	return result, nil
}

// UpdatePrefix tries to update the prefix.
// Returns OptimisticLockError if it does not succeed due to a concurrent update.
func (s *sql) UpdatePrefix(prefix Prefix) (Prefix, error) {
	oldVersion := prefix.version
	prefix.version = oldVersion + 1
	pn, err := json.Marshal(prefix.toPrefixJSON())
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to marshal prefix:%w", err)
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%w", err)
	}
	result, err := tx.Exec("SELECT prefix FROM prefixes WHERE cidr=$1 AND prefix->>'Version'=$2 FOR UPDATE", prefix.Cidr, oldVersion)
	if err != nil {
		return Prefix{}, fmt.Errorf("%w: unable to select for update prefix:%s", ErrOptimisticLockError, prefix.Cidr)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Prefix{}, err
	}
	if rows == 0 {
		// Rollback, but ignore error, if rollback is omitted, the row lock created by SELECT FOR UPDATE will not get released.
		_ = tx.Rollback()
		return Prefix{}, fmt.Errorf("%w: select for update did not effect any row", ErrOptimisticLockError)
	}
	result, err = tx.Exec("UPDATE prefixes SET prefix=$1 WHERE cidr=$2 AND prefix->>'Version'=$3", pn, prefix.Cidr, oldVersion)
	if err != nil {
		return Prefix{}, fmt.Errorf("%w: unable to update prefix:%s", ErrOptimisticLockError, prefix.Cidr)
	}
	rows, err = result.RowsAffected()
	if err != nil {
		return Prefix{}, err
	}
	if rows == 0 {
		// Rollback, but ignore error, if rollback is omitted, the row lock created by SELECT FOR UPDATE will not get released.
		_ = tx.Rollback()
		return Prefix{}, fmt.Errorf("%w: updatePrefix did not effect any row", ErrOptimisticLockError)
	}
	return prefix, tx.Commit()
}

func (s *sql) DeletePrefix(prefix Prefix) (Prefix, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%w", err)
	}
	_, err = tx.Exec("DELETE from prefixes WHERE cidr=$1", prefix.Cidr)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable delete prefix:%w", err)
	}
	return prefix, tx.Commit()
}
