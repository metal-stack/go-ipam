package ipam

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

)

type sql struct {
	db *sqlx.DB
}

func (s *sql) CreateNetwork(network *Network) (*Network, error) {
	if network.ID != "" {
		return nil, fmt.Errorf("network already created:%v", network)
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	network.ID = id.String()
	tx := s.db.MustBegin()
	n, err := json.Marshal(network)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal network:%v", err)
	}
	tx.MustExec("INSERT INTO networks (id, network) VALUES ($1, $2)", network.ID, n)
	return network, tx.Commit()
}
func (s *sql) ReadNetwork(id string) (*Network, error) {
	var result []byte
	err := s.db.Get(&result, "SELECT network FROM networks WHERE id=$1", id)
	if err != nil {
		return nil, fmt.Errorf("unable to read network:%v", err)
	}
	var network Network
	err = json.Unmarshal(result, &network)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal network:%v", err)
	}

	return &network, nil
}

func (s *sql) ReadAllNetworks() ([]*Network, error) {
	var networks [][]byte
	err := s.db.Select(&networks, "SELECT network FROM networks")
	if err != nil {
		return nil, fmt.Errorf("unable to read networks:%v", err)
	}

	var result []*Network
	for _, v := range networks {
		var net Network
		err = json.Unmarshal(v, &net)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal network:%v", err)
		}
		result = append(result, &net)
	}
	return result, nil
}
func (s *sql) UpdateNetwork(network *Network) (*Network, error) {
	tx := s.db.MustBegin()
	n, err := json.Marshal(network)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal network:%v", err)
	}
	tx.MustExec("UPDATE networks SET network=$1 WHERE id=$2", n, network.ID)
	return network, tx.Commit()
}
func (s *sql) DeleteNetwork(network *Network) (*Network, error) {
	tx := s.db.MustBegin()
	tx.MustExec("DELETE from networks WHERE id=$1", network.ID)
	return network, tx.Commit()
}

func (s *sql) prefixExists(prefix *Prefix) (*Prefix, bool) {
	pre, err := s.ReadPrefix(prefix.Cidr)
	if err != nil {
		return nil, false
	}
	if pre == nil {
		return nil, false
	}
	return pre, true
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

	var result []*Prefix
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
