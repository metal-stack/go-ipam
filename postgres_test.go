package ipam

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatasource(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		user     string
		password string
		dbname   string
		sslmode  SSLMode
		want     string
		wantErr  bool
	}{
		{
			name:     "basic, no escape",
			host:     "host",
			port:     "5432",
			user:     "user",
			password: "password",
			dbname:   "dbname",
			sslmode:  SSLModeAllow,
			want:     "postgres://user:password@host:5432/dbname?sslmode=allow",
			wantErr:  false,
		},
		{
			name:     "username and password with escape chars",
			host:     "host",
			port:     "5432",
			user:     "us@r",
			password: "pass:word",
			dbname:   "dbname",
			sslmode:  SSLModeAllow,
			want:     "postgres://us%40r:pass%3Aword@host:5432/dbname?sslmode=allow",
			wantErr:  false,
		},
		{
			name:     "spaces are not allowed in username/password",
			host:     "host",
			port:     "5432",
			user:     "us er",
			password: "pass word",
			dbname:   "dbname",
			sslmode:  SSLModeAllow,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "space allowed in dbname",
			host:     "host",
			port:     "5432",
			user:     "user",
			password: "password",
			dbname:   "db name",
			sslmode:  SSLModeAllow,
			want:     "postgres://user:password@host:5432/db%20name?sslmode=allow",
			wantErr:  false,
		},
		{
			name:     "empty password",
			host:     "host",
			port:     "5432",
			user:     "user",
			password: "",
			dbname:   "db name",
			sslmode:  SSLModeAllow,
			want:     "postgres://user:@host:5432/db%20name?sslmode=allow",
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := dataSource(tc.host, tc.port, tc.user, tc.password, tc.dbname, tc.sslmode)
			require.Equal(t, tc.wantErr, err != nil)
			require.Equal(t, tc.want, got)
		})
	}
}
