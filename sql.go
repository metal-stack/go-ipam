package ipam

import (
	"context"
	sq "database/sql" // standard interfaces for crdb compatibility
	"encoding/json"
	"fmt"
	"github.com/cockroachdb/cockroach-go/crdb" // crdb is a wrapper around the logic for issuing SQL transactions which performs retries (as required by CockroachDB)
	"github.com/jmoiron/sqlx"
)

type sql struct {
	db *sqlx.DB
}

type prefixJSON struct {
	Prefix
	AvailableChildPrefixes map[string]bool // available child prefixes of this prefix
	ChildPrefixLength      int             // the length of the child prefixes
	IPs                    map[string]bool // The ips contained in this prefix
	Version                int64           // Version is used for optimistic locking
}

func (p prefixJSON) toPrefix() Prefix {
	return Prefix{
		Cidr:                   p.Cidr,
		ParentCidr:             p.ParentCidr,
		availableChildPrefixes: p.AvailableChildPrefixes,
		childPrefixLength:      p.ChildPrefixLength,
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
		ChildPrefixLength:      p.childPrefixLength,
		IPs:                    p.ips,
		Version:                p.version,
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
		return Prefix{}, fmt.Errorf("unable to marshal prefix:%v", err)
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
	var pre prefixJSON
	err = json.Unmarshal(result, &pre)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to unmarshal prefix:%v", err)
	}

	return pre.toPrefix(), nil
}

func (s *sql) ReadAllPrefixes() ([]Prefix, error) {
	var prefixes [][]byte
	err := s.db.Select(&prefixes, "SELECT prefix FROM prefixes")
	if err != nil {
		return nil, fmt.Errorf("unable to read prefixes:%v", err)
	}

	result := []Prefix{}
	for _, v := range prefixes {
		var pre prefixJSON
		err = json.Unmarshal(v, &pre)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal prefix:%v", err)
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
		return Prefix{}, fmt.Errorf("unable to marshal prefix:%v", err)
	}

	err = crdb.ExecuteTx(context.TODO(), s.db.DB, nil, func(tx *sq.Tx) error {

		// FOR UPDATE is currently not supported for cockroach
		result, err := tx.Exec("SELECT prefix FROM prefixes WHERE cidr=$1 AND prefix->>'Version'=$2 FOR UPDATE", prefix.Cidr, oldVersion)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return NewOptimisticLockError("select for update did not effect any row")
		}
		result, err = tx.Exec("UPDATE prefixes SET prefix=$1 WHERE cidr=$2 AND prefix->>'Version'=$3", pn, prefix.Cidr, oldVersion)
		if err != nil {
			return err
		}
		rows, err = result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return NewOptimisticLockError("updatePrefix did not effect any row")
		}
		if rows > 1 {
			return NewOptimisticLockError("updatePrefix effected more than one row")
		}

		return nil
	})

	// We return the incoming prefix with incremented version instead of re-reading it from db for simplicity and performance reasons.
	return prefix, err
}

func (s *sql) DeletePrefix(prefix Prefix) (Prefix, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%v", err)
	}
	tx.MustExec("DELETE from prefixes WHERE cidr=$1", prefix.Cidr)
	return prefix, tx.Commit()
}
