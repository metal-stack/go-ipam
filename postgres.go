package ipam

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	// import for sqlx to use postgres driver
	_ "github.com/lib/pq"
)

type postgres struct {
	db *sqlx.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS networks (
	id       text PRIMARY KEY NOT NULL,
	network  JSONB
);
CREATE TABLE IF NOT EXISTS prefixes (
	cidr   text PRIMARY KEY NOT NULL,
	prefix JSONB
);
`

// NewPostgresStorage creates a new Storage which uses postgres.
func NewPostgresStorage(host, port, user, password, dbname, sslmode string) (*postgres, error) {
	db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s", host, port, user, dbname, password, sslmode))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database:%v", err)
	}
	db.MustExec(schema)
	return &postgres{
		db: db,
	}, nil
}

func (p *postgres) CreateNetwork(network *Network) (*Network, error) {
	if network.ID != "" {
		return nil, fmt.Errorf("network already created:%v", network)
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	network.ID = id.String()
	tx := p.db.MustBegin()
	n, err := json.Marshal(network)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal network:%v", err)
	}
	tx.MustExec("INSERT INTO networks (id, network) VALUES ($1, $2)", network.ID, n)
	return network, tx.Commit()
}
func (p *postgres) ReadNetwork(id string) (*Network, error) {
	var result []byte
	err := p.db.Get(&result, "SELECT network FROM networks WHERE id=$1", id)
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

func (p *postgres) ReadAllNetworks() ([]*Network, error) {
	var networks [][]byte
	err := p.db.Select(&networks, "SELECT network FROM networks")
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
func (p *postgres) UpdateNetwork(network *Network) (*Network, error) {
	tx := p.db.MustBegin()
	n, err := json.Marshal(network)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal network:%v", err)
	}
	tx.MustExec("UPDATE networks SET network=$1 WHERE id=$2", n, network.ID)
	return network, tx.Commit()
}
func (p *postgres) DeleteNetwork(network *Network) (*Network, error) {
	tx := p.db.MustBegin()
	tx.MustExec("DELETE from networks WHERE id=$1", network.ID)
	return network, tx.Commit()
}

func (p *postgres) prefixExists(prefix *Prefix) (*Prefix, bool) {
	pre, err := p.ReadPrefix(prefix.Cidr)
	if err != nil {
		return nil, false
	}
	if pre == nil {
		return nil, false
	}
	return pre, true
}

func (p *postgres) CreatePrefix(prefix *Prefix) (*Prefix, error) {
	existingPrefix, exists := p.prefixExists(prefix)
	if exists {
		return existingPrefix, nil
	}
	tx := p.db.MustBegin()
	pj, err := json.Marshal(prefix)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal prefix:%v", err)
	}
	tx.MustExec("INSERT INTO prefixes (cidr, prefix) VALUES ($1, $2)", prefix.Cidr, pj)
	return prefix, tx.Commit()
}
func (p *postgres) ReadPrefix(prefix string) (*Prefix, error) {
	var result []byte
	err := p.db.Get(&result, "SELECT prefix FROM prefixes WHERE cidr=$1", prefix)
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
func (p *postgres) ReadAllPrefixes() ([]*Prefix, error) {
	var prefixes [][]byte
	err := p.db.Select(&prefixes, "SELECT prefix FROM prefixes")
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
func (p *postgres) UpdatePrefix(prefix *Prefix) (*Prefix, error) {
	tx := p.db.MustBegin()
	pn, err := json.Marshal(prefix)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal prefix:%v", err)
	}
	tx.MustExec("UPDATE prefixes SET prefix=$1 WHERE cidr=$2", pn, prefix.Cidr)
	return prefix, tx.Commit()
}
func (p *postgres) DeletePrefix(prefix *Prefix) (*Prefix, error) {
	tx := p.db.MustBegin()
	tx.MustExec("DELETE from prefixes WHERE cidr=$1", prefix.Cidr)
	return prefix, tx.Commit()
}
