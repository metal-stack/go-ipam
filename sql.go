package ipam

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type sql struct {
	db     *sqlx.DB
	tables map[string]struct{}
}

func createTableSQL(namespace string) string {
	return fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS prefixes_%s (
	cidr   text PRIMARY KEY NOT NULL,
	prefix JSONB
);
CREATE INDEX IF NOT EXISTS prefix_idx ON prefixes_%s USING GIN(prefix);
`, namespace, namespace)
}

func (s *sql) prefixExists(ctx context.Context, prefix Prefix, namespace string) (*Prefix, bool) {
	p, err := s.ReadPrefix(ctx, prefix.Cidr, namespace)
	if err != nil {
		return nil, false
	}
	return &p, true
}

func (s *sql) CreatePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	if _, ok := s.tables[namespace]; !ok {
		if _, err := s.db.ExecContext(ctx, createTableSQL(namespace)); err != nil {
			return Prefix{}, fmt.Errorf("unable to create table:%w", err)
		}
		s.tables[namespace] = struct{}{}
	}

	existingPrefix, exists := s.prefixExists(ctx, prefix, namespace)
	if exists {
		return *existingPrefix, nil
	}
	prefix.version = int64(0)
	pj, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%w", err)
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO prefixes_"+namespace+"(cidr, prefix) VALUES ($1, $2)", prefix.Cidr, pj)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to insert prefix:%w", err)
	}
	return prefix, tx.Commit()
}

func (s *sql) ReadPrefix(ctx context.Context, prefix, namespace string) (Prefix, error) {
	var result []byte
	err := s.db.GetContext(ctx, &result, "SELECT prefix FROM prefixes_"+namespace+" WHERE cidr=$1", prefix)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read prefix:%w", err)
	}
	return fromJSON(result)
}

func (s *sql) DeleteAllPrefixes(ctx context.Context, namespace string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM prefixes_"+namespace)
	return err
}

// ReadAllPrefixes returns all known prefixes.
func (s *sql) ReadAllPrefixes(ctx context.Context, namespace string) (Prefixes, error) {
	var prefixes [][]byte
	err := s.db.SelectContext(ctx, &prefixes, "SELECT prefix FROM prefixes_"+namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to read prefixes:%w", err)
	}
	return toPrefixes(prefixes)
}

func toPrefixes(prefixes [][]byte) ([]Prefix, error) {
	result := Prefixes{}
	for _, v := range prefixes {
		pfx, err := fromJSON(v)
		if err != nil {
			return nil, err
		}
		result = append(result, pfx)
	}
	return result, nil
}

// ReadAllPrefixCidrs is cheaper that ReadAllPrefixes because it only returns the Cidrs.
func (s *sql) ReadAllPrefixCidrs(ctx context.Context, namespace string) ([]string, error) {
	cidrs := []string{}
	err := s.db.SelectContext(ctx, &cidrs, "SELECT cidr FROM prefixes_"+namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to read prefixes in namespace:%s :%w", namespace, err)
	}
	return cidrs, nil
}

// UpdatePrefix tries to update the prefix.
// Returns OptimisticLockError if it does not succeed due to a concurrent update.
func (s *sql) UpdatePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	oldVersion := prefix.version
	prefix.version = oldVersion + 1
	pn, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%w", err)
	}
	result, err := tx.ExecContext(ctx, "SELECT prefix FROM prefixes_"+namespace+" WHERE cidr=$1 AND prefix->>'Version'=$2 FOR UPDATE", prefix.Cidr, oldVersion)
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
	result, err = tx.ExecContext(ctx, "UPDATE prefixes_"+namespace+" SET prefix=$1 WHERE cidr=$2 AND prefix->>'Version'=$3", pn, prefix.Cidr, oldVersion)
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

func (s *sql) DeletePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction:%w", err)
	}
	_, err = tx.ExecContext(ctx, "DELETE from prefixes_"+namespace+" WHERE cidr=$1", prefix.Cidr)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable delete prefix:%w", err)
	}
	return prefix, tx.Commit()
}
func (s *sql) Name() string {
	return "postgres"
}
