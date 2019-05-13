package ipam

import (
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIpamer_AcquireIP(t *testing.T) {

	type fields struct {
		prefixCIDR  string
		existingips []string
	}
	tests := []struct {
		name   string
		fields fields
		want   *IP
	}{
		{
			name: "Acquire next IP regularly",
			fields: fields{
				prefixCIDR:  "192.168.1.0/24",
				existingips: []string{},
			},
			want: &IP{IP: net.ParseIP("192.168.1.1")},
		},
		{
			name: "Want next IP, network already occupied a little",
			fields: fields{
				prefixCIDR:  "192.168.2.0/30",
				existingips: []string{"192.168.2.1"},
			},
			want: &IP{IP: net.ParseIP("192.168.2.2")},
		},
		{
			name: "Want next IP, but network is full",
			fields: fields{
				prefixCIDR:  "192.168.3.0/30",
				existingips: []string{"192.168.3.1", "192.168.3.2"},
			},
			want: nil,
		},
		{
			name: "Want next IP, but network is full",
			fields: fields{
				prefixCIDR: "192.168.4.0/32",
			},
			want: nil,
		},
	}
	for _, tt := range tests {

		testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
			p, err := ipam.NewPrefix(tt.fields.prefixCIDR)
			if err != nil {
				t.Errorf("Could not create prefix: %v", err)
			}
			for _, ipString := range tt.fields.existingips {
				p.ips[ipString] = true
			}
			got, _ := ipam.AcquireIP(p)
			if tt.want == nil || got == nil {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Ipamer.AcquireIP() = %v, want %v", got, tt.want)
				}
			} else {
				if !tt.want.IP.Equal(got.IP) {
					t.Errorf("Ipamer.AcquireIP() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestIpamer_ReleaseIPFromPrefix(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/24")
		require.Nil(t, err)
		require.NotNil(t, prefix)

		err = ipam.ReleaseIPFromPrefix(nil, "1.2.3.4")
		require.NotNil(t, err)
		require.Equal(t, "prefix is nil", err.Error())

		err = ipam.ReleaseIPFromPrefix(prefix, "1.2.3.4")
		require.NotNil(t, err)
		require.Equal(t, "unable to release ip:1.2.3.4 because it is not allocated in prefix:192.168.0.0/24", err.Error())
	})
}

func TestIpamer_AcquireIPCounts(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/24")
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		// network an broadcast are blocked
		require.Equal(t, prefix.acquiredips(), uint64(2))
		ip1, err := ipam.AcquireIP(prefix)
		require.Nil(t, err)
		require.NotNil(t, ip1)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(3))
		ip2, err := ipam.AcquireIP(prefix)
		require.Nil(t, err)
		require.NotEqual(t, ip1, ip2)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(4))
		require.True(t, strings.HasPrefix(ip1.IP.String(), "192.168.0"))
		require.True(t, strings.HasPrefix(ip2.IP.String(), "192.168.0"))

		prefix, err = ipam.ReleaseIP(ip1)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(3))

		prefix, err = ipam.ReleaseIP(ip2)
		require.Nil(t, err)
		require.Equal(t, prefix.availableips(), uint64(256))
		require.Equal(t, prefix.acquiredips(), uint64(2))
	})
}

func TestIpamer_AcquireChildPrefixCounts(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
		allPrefixes, err := ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 0, len(allPrefixes))

		prefix, err := ipam.NewPrefix("192.168.0.0/20")
		require.Nil(t, err)
		require.Equal(t, prefix.availablePrefixes(), uint64(0))
		require.Equal(t, prefix.acquiredPrefixes(), uint64(0))
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(0))

		usage := prefix.Usage()
		require.Equal(t, "ip:2/4096", usage.String())

		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 1, len(allPrefixes))

		c1, err := ipam.AcquireChildPrefix(prefix, 22)
		require.Nil(t, err)
		require.NotNil(t, c1)
		require.Equal(t, prefix.availablePrefixes(), uint64(4))
		require.Equal(t, prefix.acquiredPrefixes(), uint64(1))
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(1))

		usage = prefix.Usage()
		require.Equal(t, "ip:2/4096 prefix:1/4", usage.String())

		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 2, len(allPrefixes))

		c2, err := ipam.AcquireChildPrefix(prefix, 22)
		require.Nil(t, err)
		require.NotNil(t, c2)
		require.Equal(t, prefix.availablePrefixes(), uint64(4))
		require.Equal(t, prefix.acquiredPrefixes(), uint64(2))
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(2))
		require.True(t, strings.HasSuffix(c1.Cidr, "/22"))
		require.True(t, strings.HasSuffix(c2.Cidr, "/22"))
		require.True(t, strings.HasPrefix(c1.Cidr, "192.168."))
		require.True(t, strings.HasPrefix(c2.Cidr, "192.168."))
		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 3, len(allPrefixes))

		err = ipam.ReleaseChildPrefix(c1)
		prefix = ipam.PrefixFrom(prefix.Cidr)

		require.Nil(t, err)
		require.Equal(t, uint64(4), prefix.availablePrefixes())
		require.Equal(t, uint64(1), prefix.acquiredPrefixes())
		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 2, len(allPrefixes))

		err = ipam.ReleaseChildPrefix(c2)
		prefix = ipam.PrefixFrom(prefix.Cidr)

		require.Nil(t, err)
		require.Equal(t, uint64(4), prefix.availablePrefixes())
		require.Equal(t, uint64(0), prefix.acquiredPrefixes())
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(0))
		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 1, len(allPrefixes))

		err = ipam.ReleaseChildPrefix(c1)
		require.Errorf(t, err, "unable to release prefix %s:delete prefix:%s not found", c1.Cidr)

		c3, err := ipam.AcquireChildPrefix(prefix, 22)
		require.Nil(t, err)
		require.NotNil(t, c2)

		ip1, err := ipam.AcquireIP(c3)
		require.Nil(t, err)
		require.NotNil(t, ip1)

		err = ipam.ReleaseChildPrefix(c3)
		require.Errorf(t, err, "prefix %s has ips, deletion not possible", c3.Cidr)

		c3, err = ipam.ReleaseIP(ip1)
		require.Nil(t, err)

		err = ipam.ReleaseChildPrefix(c3)
		require.Nil(t, err)

		allPrefixes, err = ipam.storage.ReadAllPrefixes()
		require.Nil(t, err)
		require.Equal(t, 1, len(allPrefixes))
	})
}

func TestIpamer_AcquireChildPrefix(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/20")
		require.Nil(t, err)
		require.Equal(t, prefix.availablePrefixes(), uint64(0))
		require.Equal(t, prefix.acquiredPrefixes(), uint64(0))

		// Same length
		cp, err := ipam.AcquireChildPrefix(prefix, 20)
		require.NotNil(t, err)
		require.Equal(t, "given length:20 is smaller or equal of prefix length:20", err.Error())
		require.Nil(t, cp)

		// working length
		cp, err = ipam.AcquireChildPrefix(prefix, 21)
		require.Nil(t, err)
		require.NotNil(t, cp)
		require.True(t, strings.HasPrefix(cp.Cidr, "192.168."))
		require.True(t, strings.HasSuffix(cp.Cidr, "/21"))
		require.Equal(t, prefix.Cidr, cp.ParentCidr)

		// different length
		cp, err = ipam.AcquireChildPrefix(prefix, 22)
		require.NotNil(t, err)
		require.Equal(t, "given length:22 is not equal to existing child prefix length:21", err.Error())
		require.Nil(t, cp)

		// No more ChildPrefixes
		cp, err = ipam.AcquireChildPrefix(prefix, 21)
		require.Nil(t, err)
		require.NotNil(t, cp)
		cp, err = ipam.AcquireChildPrefix(prefix, 21)
		require.NotNil(t, err)
		require.Equal(t, "no more child prefixes contained in prefix pool", err.Error())
		require.Nil(t, cp)

		// Prefix has ips
		p2, err := ipam.NewPrefix("10.0.0.0/24")
		require.Nil(t, err)
		require.Equal(t, p2.availablePrefixes(), uint64(0))
		require.Equal(t, p2.acquiredPrefixes(), uint64(0))
		ip, err := ipam.AcquireIP(p2)
		require.Nil(t, err)
		require.NotNil(t, ip)
		cp2, err := ipam.AcquireChildPrefix(p2, 25)
		require.NotNil(t, err)
		require.Equal(t, "prefix 10.0.0.0/24 has ips, acquire child prefix not possible", err.Error())
		require.Nil(t, cp2)

		// Prefix has Childs, AcquireIP wont work
		p3, err := ipam.NewPrefix("172.17.0.0/24")
		require.Nil(t, err)
		require.Equal(t, p3.availablePrefixes(), uint64(0))
		require.Equal(t, p3.acquiredPrefixes(), uint64(0))
		cp3, err := ipam.AcquireChildPrefix(p3, 25)
		require.Nil(t, err)
		require.NotNil(t, cp3)
		ip, err = ipam.AcquireIP(p3)
		require.NotNil(t, err)
		require.Equal(t, "prefix 172.17.0.0/24 has childprefixes, acquire ip not possible", err.Error())
		require.Nil(t, ip)

		// Release Parent Prefix must not work
		err = ipam.ReleaseChildPrefix(p3)
		require.NotNil(t, err)
		require.Equal(t, "prefix 172.17.0.0/24 is no child prefix", err.Error())
	})
}

func TestPrefix_Availableips(t *testing.T) {

	tests := []struct {
		name string
		Cidr string
		want uint64
	}{
		{
			name: "large",
			Cidr: "192.168.0.0/20",
			want: 4096,
		},
		{
			name: "small",
			Cidr: "192.168.0.0/24",
			want: 256,
		},
		{
			name: "smaller",
			Cidr: "192.168.0.0/25",
			want: 128,
		},
		{
			name: "smaller",
			Cidr: "192.168.0.0/30",
			want: 4,
		},
		// {
		// 	name: "small IPv6",
		// 	Cidr: "2001:16b8:2d6a:6900:48d2:14a3:80ae:e797/64",
		// 	want: 4,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Prefix{
				Cidr: tt.Cidr,
			}
			if got := p.availableips(); got != tt.want {
				t.Errorf("Prefix.Availableips() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIpamer_PrefixesOverlapping(t *testing.T) {

	tests := []struct {
		name             string
		existingPrefixes []string
		newPrefixes      []string
		wantErr          bool
		errorString      string
	}{
		{
			name:             "simple",
			existingPrefixes: []string{"192.168.0.0/24"},
			newPrefixes:      []string{"192.168.1.0/24"},
			wantErr:          false,
			errorString:      "",
		},
		{
			name:             "one overlap",
			existingPrefixes: []string{"192.168.0.0/24", "192.168.1.0/24"},
			newPrefixes:      []string{"192.168.1.0/24"},
			wantErr:          true,
			errorString:      "192.168.1.0/24 overlaps 192.168.1.0/24",
		},
		{
			name:             "one overlap",
			existingPrefixes: []string{"192.168.0.0/24", "192.168.1.0/24"},
			newPrefixes:      []string{"192.168.0.0/23"},
			wantErr:          true,
			errorString:      "192.168.0.0/23 overlaps 192.168.0.0/24",
		},
		{
			name:             "one overlap",
			existingPrefixes: []string{"192.168.0.0/23", "192.168.2.0/23"},
			newPrefixes:      []string{"192.168.3.0/24"},
			wantErr:          true,
			errorString:      "192.168.3.0/24 overlaps 192.168.2.0/23",
		},
		{
			name:             "one overlap",
			existingPrefixes: []string{"192.168.128.0/25"},
			newPrefixes:      []string{"192.168.128.0/27"},
			wantErr:          true,
			errorString:      "192.168.128.0/27 overlaps 192.168.128.0/25",
		},
	}
	for _, tt := range tests {
		testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
			for _, ep := range tt.existingPrefixes {
				p, err := ipam.NewPrefix(ep)
				if err != nil {
					t.Errorf("Newprefix on ExistingPrefix failed:%v", err)
				}
				if p == nil {
					t.Errorf("Newprefix on ExistingPrefix returns nil")
				}
			}
			err := ipam.PrefixesOverlapping(tt.existingPrefixes, tt.newPrefixes)
			if tt.wantErr && err == nil {
				t.Errorf("Ipamer.PrefixesOverlapping() expected error but err was nil")
			}
			if tt.wantErr && err != nil && err.Error() != tt.errorString {
				t.Errorf("Ipamer.PrefixesOverlapping() error = %v, wantErr %v, errorString = %v", err, tt.wantErr, tt.errorString)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Ipamer.PrefixesOverlapping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIpamer_NewPrefix(t *testing.T) {

	tests := []struct {
		name        string
		cidr        string
		wantErr     bool
		errorString string
	}{
		{
			name:    "valid Prefix",
			cidr:    "192.168.0.0/24",
			wantErr: false,
		},
		{
			name:        "invalid Prefix",
			cidr:        "192.168.0.0/33",
			wantErr:     true,
			errorString: "unable to parse cidr:192.168.0.0/33 invalid CIDR address: 192.168.0.0/33",
		},
	}
	for _, tt := range tests {

		testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
			got, err := ipam.NewPrefix(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Ipamer.NewPrefix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (err != nil) && tt.errorString != err.Error() {
				t.Errorf("Ipamer.NewPrefix() error = %v, errorString %v", err, tt.errorString)
				return
			}

			if err != nil {
				return
			}
			if !reflect.DeepEqual(got.Cidr, tt.cidr) {
				t.Errorf("Ipamer.NewPrefix() = %v, want %v", got.Cidr, tt.cidr)
			}
		})
	}
}

func TestIpamer_DeletePrefix(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
		prefix, err := ipam.NewPrefix("192.168.0.0/20")
		require.Nil(t, err)
		require.Equal(t, prefix.availablePrefixes(), uint64(0))
		require.Equal(t, prefix.acquiredPrefixes(), uint64(0))
		require.Equal(t, prefix.Usage().AcquiredPrefixes, uint64(0))

		ip, err := ipam.AcquireIP(prefix)
		require.Nil(t, err)
		require.NotNil(t, ip)

		_, err = ipam.DeletePrefix(prefix.Cidr)
		require.NotNil(t, err)
		require.Equal(t, "prefix 192.168.0.0/20 has ips, delete prefix not possible", err.Error())
	})
}

func TestIpamer_PrefixFrom(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
		prefix := ipam.PrefixFrom("192.168.0.0/20")
		require.Nil(t, prefix)

		prefix, err := ipam.NewPrefix("192.168.0.0/20")
		require.Nil(t, err)
		require.NotNil(t, prefix)

		prefix = ipam.PrefixFrom("192.168.0.0/20")
		require.NotNil(t, prefix)
	})
}

func TestIpamerAcquireIP(t *testing.T) {

	testWithBackends(t, func(t *testing.T, ipam *Ipamer) {
		cidr := "10.0.0.0/16"
		p, err := ipam.NewPrefix(cidr)
		if err != nil {
			panic(err)
		}
		for n := 0; n < 10; n++ {
			if len(p.ips) != 2 {
				t.Fatalf("expected 2 ips in prefix, got %d", len(p.ips))
			}
			ip, err := ipam.AcquireIP(p)
			if err != nil {
				panic(err)
			}
			if ip == nil {
				panic("IP nil")
			}
			p, err = ipam.ReleaseIP(ip)
			if err != nil {
				panic(err)
			}
		}
		_, err = ipam.DeletePrefix(cidr)
		if err != nil {
			t.Errorf("error deleting prefix:%v", err)
		}
	})
}

func NewPostgres() (*sql, error) {
	return NewPostgresStorage("localhost", "5433", "postgres", "password", "postgres", "disable")
}

// interface for impls that support cleaning before each testrun
type Cleanable interface {
	cleanup() error
}

// extended sql interface
type ExtendedSql struct {
	*sql
}

func NewPostgresWithCleanup() (*ExtendedSql, error) {
	s, err := NewPostgres()
	if err != nil {
		return nil, err
	}

	ext := &ExtendedSql{
		s,
	}

	return ext, nil
}

// cleanup database before test
func (e *ExtendedSql) cleanup() error {
	tx := e.db.MustBegin()
	_, err := e.db.Exec("TRUNCATE TABLE prefixes")
	if err != nil {
		return err
	}
	return tx.Commit()
}

type testMethod func(t *testing.T, ipam *Ipamer)

func testWithBackends(t *testing.T, fn testMethod) {
	for _, storageProvider := range storageProviders() {

		storage := storageProvider.provide()

		if tp, ok := storage.(Cleanable); ok {
			err := tp.cleanup()
			if err != nil {
				t.Errorf("error cleaing up, %v", err)
			}
		}

		ipamer := NewWithStorage(storage)
		testName := storageProvider.name

		t.Run(testName, func(t *testing.T) {
			fn(t, ipamer)
		})
	}
}

type provide func() Storage
type StorageProvider struct {
	name    string
	provide provide
}

func storageProviders() []StorageProvider {
	return []StorageProvider{
		{
			name: "Memory",
			provide: func() Storage {
				return NewMemory()
			},
		},
		{
			name: "Postgres",
			provide: func() Storage {
				storage, err := NewPostgresWithCleanup()
				if err != nil {
					panic("error getting postgres storage")
				}
				return storage
			},
		},
	}
}
