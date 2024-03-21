package ipam

import (
	"context"
	"testing"

	"time"

	"github.com/stretchr/testify/require"
)

func Test_sql_prefixExists(t *testing.T) {
	ctx := context.Background()
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		// Existing Prefix
		prefix := Prefix{Cidr: "10.0.0.0/16"}
		p, err := db.CreatePrefix(ctx, prefix, defaultNamespace)
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Equal(t, prefix.Cidr, p.Cidr)
		got, exists := db.prefixExists(ctx, prefix, defaultNamespace)
		require.True(t, exists)
		require.Equal(t, got.Cidr, prefix.Cidr)

		// NonExisting Prefix
		notExistingPrefix := Prefix{Cidr: "10.0.0.0/8"}
		got, exists = db.prefixExists(ctx, notExistingPrefix, defaultNamespace)
		require.False(t, exists)
		require.Nil(t, got)

		// Delete Existing Prefix
		_, err = db.DeletePrefix(ctx, prefix, defaultNamespace)
		require.NoError(t, err)
		got, exists = db.prefixExists(ctx, prefix, defaultNamespace)
		require.False(t, exists)
		require.Nil(t, got)
	})
}

func Test_sql_CreatePrefix(t *testing.T) {
	ctx := context.Background()
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		// Existing Prefix
		prefix := Prefix{Cidr: "11.0.0.0/16"}
		got, exists := db.prefixExists(ctx, prefix, defaultNamespace)
		require.False(t, exists)
		require.Nil(t, got)
		p, err := db.CreatePrefix(ctx, prefix, defaultNamespace)
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Equal(t, prefix.Cidr, p.Cidr)
		got, exists = db.prefixExists(ctx, prefix, defaultNamespace)
		require.True(t, exists)
		require.Equal(t, got.Cidr, prefix.Cidr)

		// Duplicate Prefix
		p, err = db.CreatePrefix(ctx, prefix, defaultNamespace)
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Equal(t, prefix.Cidr, p.Cidr)

		ps, err := db.ReadAllPrefixCidrs(ctx, defaultNamespace)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Len(t, ps, 1)
	})
}

func Test_sql_ReadPrefix(t *testing.T) {
	ctx := context.Background()
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		// Prefix
		p, err := db.ReadPrefix(ctx, "12.0.0.0/8", "a")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNamespaceDoesNotExist)
		require.Empty(t, p)

		prefix := Prefix{Cidr: "12.0.0.0/16"}

		// Create Namespace
		err = db.CreateNamespace(ctx, "a")
		require.NoError(t, err)

		p, err = db.CreatePrefix(ctx, prefix, "a")
		require.NoError(t, err)
		require.NotNil(t, p)

		p, err = db.ReadPrefix(ctx, "12.0.0.0/16", "a")
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Equal(t, "12.0.0.0/16", p.Cidr)
	})
}

func Test_sql_ReadAllPrefix(t *testing.T) {
	ctx := context.Background()
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		// no Prefixes
		ps, err := db.ReadAllPrefixCidrs(ctx, defaultNamespace)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Empty(t, ps)

		// One Prefix
		prefix := Prefix{Cidr: "12.0.0.0/16"}
		p, err := db.CreatePrefix(ctx, prefix, defaultNamespace)
		require.NoError(t, err)
		require.NotNil(t, p)
		ps, err = db.ReadAllPrefixCidrs(ctx, defaultNamespace)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Len(t, ps, 1)

		// no Prefixes again
		_, err = db.DeletePrefix(ctx, prefix, defaultNamespace)
		require.NoError(t, err)
		ps, err = db.ReadAllPrefixCidrs(ctx, defaultNamespace)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Empty(t, ps)
	})
}

func Test_sql_CreateNamespace(t *testing.T) {
	ctx := context.Background()
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)
		{
			// Create a namespace with special characters in name
			namespace := "%u6c^qi$u%tSqhQTcjR!zZHNvMB$3XJd"
			err := db.CreateNamespace(ctx, namespace)
			require.NoError(t, err)

			err = db.DeleteNamespace(ctx, namespace)
			require.NoError(t, err)
		}
		{
			// Create a long namespace name
			namespace := "d4546731-6056-4b48-80e9-ef924ca7f651"
			err := db.CreateNamespace(ctx, namespace)
			require.NoError(t, err)

			err = db.DeleteNamespace(ctx, namespace)
			require.NoError(t, err)
		}
		{
			// Create a namespace with a name that is too long
			namespace := "d4546731-6056-4b48-80e9-ef924ca7f651d4546731-6056-4b48-80e9-ef924ca7f651d4546731-6056-4b48-80e9-ef924ca7f651d4546731-6056-4b48-80e9-ef924ca7f651"
			err := db.CreateNamespace(ctx, namespace)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrNameTooLong)
		}
	})
}

func Test_ConcurrentAcquirePrefix(t *testing.T) {
	ctx := context.Background()
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		ipamer := NewWithStorage(db)

		const parentCidr = "1.0.0.0/16"
		_, err := ipamer.NewPrefix(ctx, parentCidr)
		require.NoError(t, err)

		count := 20
		prefixes := make(chan string)
		for range count {
			// FIXME migrate to errgroup
			go acquirePrefix(t, ctx, db, parentCidr, prefixes) // nolint:testifylint
		}

		prefixMap := make(map[string]bool)
		for range count {
			p := <-prefixes
			_, duplicate := prefixMap[p]
			if duplicate {
				t.Errorf("prefix:%s already acquired", p)
			}
			prefixMap[p] = true
		}
	})
}

func acquirePrefix(t *testing.T, ctx context.Context, db *sql, cidr string, prefixes chan string) {
	require.NotNil(t, db)
	ipamer := NewWithStorage(db)

	var cp *Prefix
	var err error
	for cp == nil {
		cp, err = ipamer.AcquireChildPrefix(ctx, cidr, 26)
		if err != nil {
			t.Error(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	prefixes <- cp.String()
}

func Test_ConcurrentAcquireIP(t *testing.T) {
	ctx := context.Background()
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		ipamer := NewWithStorage(db)

		const parentCidr = "2.7.0.0/16"
		_, err := ipamer.NewPrefix(ctx, parentCidr)
		require.NoError(t, err)

		count := 30
		ips := make(chan string)
		for range count {
			// FIXME migrate to errgroup
			go acquireIP(t, ctx, db, parentCidr, ips) // nolint:testifylint
		}

		ipMap := make(map[string]bool)
		for range count {
			p := <-ips
			_, duplicate := ipMap[p]
			if duplicate {
				t.Errorf("prefix:%s already acquired", p)
			}
			ipMap[p] = true
		}
	})
}

func acquireIP(t *testing.T, ctx context.Context, db *sql, prefix string, ips chan string) {
	require.NotNil(t, db)
	ipamer := NewWithStorage(db)

	var ip *IP
	var err error
	for ip == nil {
		ip, err = ipamer.AcquireIP(ctx, prefix)
		if err != nil {
			t.Error(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	ips <- ip.IP.String()
}
