package ipam

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// tsql allows explicit transaction boundaries
type tsql struct {
	db   *sqlx.DB
	tx   *sqlx.Tx
	inTX bool
}

func (t *tsql) Begin() error {
	if t.inTX {
		return fmt.Errorf("unable to start a new transaction in a existing transaction")
	}
	tx, err := t.db.Beginx()
	if err != nil {
		return fmt.Errorf("unable to start transaction:%v", err)
	}
	t.tx = tx
	t.inTX = true
	return nil
}

func (t *tsql) Commit() error {
	t.inTX = false
	return t.tx.Commit()
}

func (t *tsql) Rollback() error {
	return t.tx.Rollback()
}

func (t *tsql) CreatePrefix(prefix *Prefix) (*Prefix, error) {
	if !t.inTX {
		return nil, fmt.Errorf("not in transaction")
	}
	existingPrefix, exists := t.prefixExists(prefix)
	if exists {
		return existingPrefix, nil
	}
	prefix.version = int64(0)
	pj, err := json.Marshal(prefix.toPrefixJSON())
	if err != nil {
		return nil, fmt.Errorf("unable to marshal prefix:%v", err)
	}
	t.tx.MustExec("INSERT INTO prefixes (cidr, prefix) VALUES ($1, $2)", prefix.Cidr, pj)
	return prefix, nil
}

func (t *tsql) ReadPrefix(prefix string) (*Prefix, error) {
	var result []byte
	err := t.db.Get(&result, "SELECT prefix FROM prefixes WHERE cidr=$1", prefix)
	if err != nil {
		return nil, fmt.Errorf("unable to read prefix:%v", err)
	}
	var pre prefixJSON
	err = json.Unmarshal(result, &pre)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal prefix:%v", err)
	}

	return pre.toPrefix(), nil
}

func (t *tsql) ReadAllPrefixes() ([]*Prefix, error) {
	var prefixes [][]byte
	err := t.db.Select(&prefixes, "SELECT prefix FROM prefixes")
	if err != nil {
		return nil, fmt.Errorf("unable to read prefixes:%v", err)
	}

	result := []*Prefix{}
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

func (t *tsql) UpdatePrefix(prefix *Prefix) (*Prefix, error) {
	oldVersion := prefix.version
	prefix.version = oldVersion + 1
	pn, err := json.Marshal(prefix.toPrefixJSON())
	if err != nil {
		return nil, fmt.Errorf("unable to marshal prefix:%v", err)
	}
	if !t.inTX {
		return nil, fmt.Errorf("not in transaction")
	}
	result := t.tx.MustExec("SELECT prefix FROM prefixes WHERE cidr=$1 AND prefix->>'Version'=$2 FOR UPDATE", prefix.Cidr, oldVersion)
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		err := t.Rollback()
		return nil, fmt.Errorf("select for update did not effect any row, %v", err)
	}
	result = t.tx.MustExec("UPDATE prefixes SET prefix=$1 WHERE cidr=$2 AND prefix->>'Version'=$3", pn, prefix.Cidr, oldVersion)
	rows, err = result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		err := t.Rollback()
		return nil, fmt.Errorf("updatePrefix did not effect any row, %v", err)
	}
	return prefix, nil
}

func (t *tsql) DeletePrefix(prefix *Prefix) (*Prefix, error) {
	if !t.inTX {
		return nil, fmt.Errorf("not in transaction")
	}
	t.tx.MustExec("DELETE from prefixes WHERE cidr=$1", prefix.Cidr)
	return prefix, nil
}

func (t *tsql) prefixExists(prefix *Prefix) (*Prefix, bool) {
	p, err := t.ReadPrefix(prefix.Cidr)
	if err != nil {
		return nil, false
	}
	if p == nil {
		return nil, false
	}
	return p, true
}
