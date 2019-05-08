package ipam

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type sql struct {
	db *sqlx.DB
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
