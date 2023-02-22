package ipam

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
)

type sql struct {
	db          *sqlx.DB
	maxIdLength int
	tables      sync.Map
}

func createTableSQL(namespace string) string {
	return fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	cidr   text PRIMARY KEY NOT NULL,
	prefix JSONB
);
CREATE INDEX IF NOT EXISTS prefix_idx ON %s USING GIN(prefix);
`, getTableName(namespace), getTableName(namespace))
}

func getTableName(namespace string) string {
	if namespace == defaultNamespace {
		return "prefixes"
	}
	return fmt.Sprintf("\"prefixes_%s\"", namespace)
}

func (s *sql) prefixExists(ctx context.Context, prefix Prefix, namespace string) (*Prefix, bool) {
	p, err := s.ReadPrefix(ctx, prefix.Cidr, namespace)
	if err != nil {
		return nil, false
	}
	return &p, true
}

func (s *sql) checkNamespaceExists(ctx context.Context, namespace string) error {
	if _, ok := s.tables.Load(namespace); ok {
		return nil
	}
	// populate namespaces from sql
	if _, err := s.ListNamespaces(ctx); err != nil {
		return err
	}
	if _, ok := s.tables.Load(namespace); !ok {
		return ErrNamespaceDoesNotExist
	}
	return nil
}

func (s *sql) CreatePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	if err := s.checkNamespaceExists(ctx, namespace); err != nil {
		return Prefix{}, err
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
	_, err = tx.ExecContext(ctx, "INSERT INTO "+getTableName(namespace)+"(cidr, prefix) VALUES ($1, $2)", prefix.Cidr, pj)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to insert prefix:%w", err)
	}
	return prefix, tx.Commit()
}

func (s *sql) ReadPrefix(ctx context.Context, prefix, namespace string) (Prefix, error) {
	if err := s.checkNamespaceExists(ctx, namespace); err != nil {
		return Prefix{}, err
	}
	var result []byte
	err := s.db.GetContext(ctx, &result, "SELECT prefix FROM "+getTableName(namespace)+" WHERE cidr=$1", prefix)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read prefix:%w", err)
	}
	return fromJSON(result)
}

func (s *sql) DeleteAllPrefixes(ctx context.Context, namespace string) error {
	if err := s.checkNamespaceExists(ctx, namespace); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM "+getTableName(namespace))
	return err
}

// ReadAllPrefixes returns all known prefixes.
func (s *sql) ReadAllPrefixes(ctx context.Context, namespace string) (Prefixes, error) {
	if err := s.checkNamespaceExists(ctx, namespace); err != nil {
		return nil, err
	}
	var prefixes [][]byte
	err := s.db.SelectContext(ctx, &prefixes, "SELECT prefix FROM "+getTableName(namespace))
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
	if err := s.checkNamespaceExists(ctx, namespace); err != nil {
		return nil, err
	}
	cidrs := []string{}
	err := s.db.SelectContext(ctx, &cidrs, "SELECT cidr FROM "+getTableName(namespace))
	if err != nil {
		return nil, fmt.Errorf("unable to read prefixes in namespace:%s :%w", namespace, err)
	}
	return cidrs, nil
}

// UpdatePrefix tries to update the prefix.
// Returns OptimisticLockError if it does not succeed due to a concurrent update.
func (s *sql) UpdatePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	if err := s.checkNamespaceExists(ctx, namespace); err != nil {
		return Prefix{}, err
	}
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
	result, err := tx.ExecContext(ctx, "SELECT prefix FROM "+getTableName(namespace)+" WHERE cidr=$1 AND prefix->>'Version'=$2 FOR UPDATE", prefix.Cidr, oldVersion)
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
	result, err = tx.ExecContext(ctx, "UPDATE "+getTableName(namespace)+" SET prefix=$1 WHERE cidr=$2 AND prefix->>'Version'=$3", pn, prefix.Cidr, oldVersion)
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
	if err := s.checkNamespaceExists(ctx, namespace); err != nil {
		return Prefix{}, err
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to start transaction: %w", err)
	}
	_, err = tx.ExecContext(ctx, "DELETE from "+getTableName(namespace)+" WHERE cidr=$1", prefix.Cidr)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable delete prefix: %w", err)
	}
	return prefix, tx.Commit()
}
func (s *sql) Name() string {
	return "postgres"
}

func (s *sql) CreateNamespace(ctx context.Context, namespace string) error {
	if len(namespace) > s.maxIdLength {
		return ErrNameTooLong
	}
	if _, ok := s.tables.Load(namespace); !ok {
		if _, err := s.db.ExecContext(ctx, createTableSQL(namespace)); err != nil {
			return fmt.Errorf("unable to create table: %w", err)
		}
		s.tables.Store(namespace, struct{}{})
	}
	return nil
}

func (s *sql) ListNamespaces(ctx context.Context) ([]string, error) {
	var result []string
	if err := s.db.SelectContext(ctx, &result, "SELECT table_name FROM information_schema.tables WHERE table_name LIKE 'prefix%'"); err != nil {
		return nil, fmt.Errorf("unable to get tables: %w", err)
	}
	for i := range result {
		if result[i] == "prefixes" {
			result[i] = defaultNamespace
		} else {
			result[i] = strings.TrimPrefix(result[i], "prefixes_")
		}
		s.tables.Store(result[i], struct{}{})
	}
	return result, nil
}

func (s *sql) DeleteNamespace(ctx context.Context, namespace string) error {
	if err := s.checkNamespaceExists(ctx, namespace); err != nil {
		return err
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("unable to start transaction: %w", err)
	}
	_, err = tx.ExecContext(ctx, "DROP TABLE "+getTableName(namespace))
	if err != nil {
		return fmt.Errorf("unable delete prefix:%w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.tables.Delete(namespace)
	return nil
}
