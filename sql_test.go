package ipam

import (
	"testing"

	"time"

	"github.com/stretchr/testify/require"
)

func Test_sql_prefixExists(t *testing.T) {
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		// Existing Prefix
		prefix := Prefix{Cidr: "10.0.0.0/16"}
		p, err := db.CreatePrefix(prefix)
		require.Nil(t, err)
		require.NotNil(t, p)
		require.Equal(t, prefix.Cidr, p.Cidr)
		got, exists := db.prefixExists(prefix)
		require.True(t, exists)
		require.Equal(t, got.Cidr, prefix.Cidr)

		// NonExisting Prefix
		notExistingPrefix := Prefix{Cidr: "10.0.0.0/8"}
		got, exists = db.prefixExists(notExistingPrefix)
		require.False(t, exists)
		require.Nil(t, got)

		// Delete Existing Prefix
		_, err = db.DeletePrefix(prefix)
		require.Nil(t, err)
		got, exists = db.prefixExists(prefix)
		require.False(t, exists)
		require.Nil(t, got)
	})
}

func Test_sql_CreatePrefix(t *testing.T) {
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		// Existing Prefix
		prefix := Prefix{Cidr: "11.0.0.0/16"}
		got, exists := db.prefixExists(prefix)
		require.False(t, exists)
		require.Nil(t, got)
		p, err := db.CreatePrefix(prefix)
		require.Nil(t, err)
		require.NotNil(t, p)
		require.Equal(t, prefix.Cidr, p.Cidr)
		got, exists = db.prefixExists(prefix)
		require.True(t, exists)
		require.Equal(t, got.Cidr, prefix.Cidr)

		// Duplicate Prefix
		p, err = db.CreatePrefix(prefix)
		require.Nil(t, err)
		require.NotNil(t, p)
		require.Equal(t, prefix.Cidr, p.Cidr)

		ps, err := db.ReadAllPrefixes()
		require.Nil(t, err)
		require.NotNil(t, ps)
		require.Equal(t, 1, len(ps))
	})
}

func Test_sql_ReadPrefix(t *testing.T) {
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		// Prefix
		p, err := db.ReadPrefix("12.0.0.0/8")
		require.NotNil(t, err)
		require.Equal(t, "unable to read prefix:sql: no rows in result set", err.Error())
		require.Empty(t, p)

		prefix := Prefix{Cidr: "12.0.0.0/16"}
		p, err = db.CreatePrefix(prefix)
		require.Nil(t, err)
		require.NotNil(t, p)

		p, err = db.ReadPrefix("12.0.0.0/16")
		require.Nil(t, err)
		require.NotNil(t, p)
		require.Equal(t, "12.0.0.0/16", p.Cidr)
	})
}

func Test_sql_ReadAllPrefix(t *testing.T) {
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		// no Prefixes
		ps, err := db.ReadAllPrefixes()
		require.Nil(t, err)
		require.NotNil(t, ps)
		require.Equal(t, 0, len(ps))

		// One Prefix
		prefix := Prefix{Cidr: "12.0.0.0/16"}
		p, err := db.CreatePrefix(prefix)
		require.Nil(t, err)
		require.NotNil(t, p)
		ps, err = db.ReadAllPrefixes()
		require.Nil(t, err)
		require.NotNil(t, ps)
		require.Equal(t, 1, len(ps))

		// no Prefixes again
		_, err = db.DeletePrefix(prefix)
		require.Nil(t, err)
		ps, err = db.ReadAllPrefixes()
		require.Nil(t, err)
		require.NotNil(t, ps)
		require.Equal(t, 0, len(ps))
	})
}

func Test_sql_UpdatePrefix(t *testing.T) {
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		// Prefix
		prefix := Prefix{Cidr: "13.0.0.0/16", ParentCidr: "13.0.0.0/8"}
		p, err := db.CreatePrefix(prefix)
		require.Nil(t, err)
		require.NotNil(t, p)

		// Check if present
		p, err = db.ReadPrefix("13.0.0.0/16")
		require.Nil(t, err)
		require.NotNil(t, p)
		require.Equal(t, "13.0.0.0/16", p.Cidr)
		require.Equal(t, "13.0.0.0/8", p.ParentCidr)

		// Modify
		prefix.ParentCidr = "13.0.0.0/12"
		p, err = db.UpdatePrefix(prefix)
		require.Nil(t, err)
		require.NotNil(t, p)
		p, err = db.ReadPrefix("13.0.0.0/16")
		require.Nil(t, err)
		require.NotNil(t, p)
		require.Equal(t, "13.0.0.0/16", p.Cidr)
		require.Equal(t, "13.0.0.0/12", p.ParentCidr)
	})
}

func Test_ConcurrentAcquirePrefix(t *testing.T) {
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		ipamer := NewWithStorage(db)

		const parentCidr = "1.0.0.0/16"
		_, err := ipamer.NewPrefix(parentCidr)
		require.Nil(t, err)

		count := 20
		prefixes := make(chan string)
		for i := 0; i < count; i++ {
			go acquirePrefix(t, db, parentCidr, prefixes)
		}

		prefixMap := make(map[string]bool)
		for i := 0; i < count; i++ {
			p := <-prefixes
			_, duplicate := prefixMap[p]
			if duplicate {
				t.Errorf("prefix:%s already acquired", p)
			}
			prefixMap[p] = true
		}
	})
}

func acquirePrefix(t *testing.T, db *sql, cidr string, prefixes chan string) {
	require.NotNil(t, db)
	ipamer := NewWithStorage(db)

	var cp *Prefix
	var err error
	for cp == nil {
		cp, err = ipamer.AcquireChildPrefix(cidr, 26)
		if err != nil {
			t.Error(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	prefixes <- cp.String()
}

func Test_ConcurrentAcquireIP(t *testing.T) {
	testWithSQLBackends(t, func(t *testing.T, db *sql) {
		require.NotNil(t, db)

		ipamer := NewWithStorage(db)

		const parentCidr = "2.7.0.0/16"
		_, err := ipamer.NewPrefix(parentCidr)
		require.Nil(t, err)

		count := 30
		ips := make(chan string)
		for i := 0; i < count; i++ {
			go acquireIP(t, db, parentCidr, ips)
		}

		ipMap := make(map[string]bool)
		for i := 0; i < count; i++ {
			p := <-ips
			_, duplicate := ipMap[p]
			if duplicate {
				t.Errorf("prefix:%s already acquired", p)
			}
			ipMap[p] = true
		}
	})
}

func acquireIP(t *testing.T, db *sql, prefix string, ips chan string) {
	require.NotNil(t, db)
	ipamer := NewWithStorage(db)

	var ip *IP
	var err error
	for ip == nil {
		ip, err = ipamer.AcquireIP(prefix)
		if err != nil {
			t.Error(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	ips <- ip.IP.String()
}
