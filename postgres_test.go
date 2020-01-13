package ipam

import (
	"testing"

	_ "github.com/lib/pq"
)

func Test_dataSource(t *testing.T) {
	type args struct {
		host     string
		port     string
		user     string
		password string
		dbname   string
		ssl      bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple",
			args: args{
				host:     "localhost",
				port:     "5432",
				user:     "root",
				password: "geheim",
				dbname:   "default",
				ssl:      false,
			},
			want: "postgres://root:geheim@localhost:5432/default?sslmode=disable",
		},
		{
			name: "simple with ssl",
			args: args{
				host:     "localhost",
				port:     "5432",
				user:     "root",
				password: "geheim",
				dbname:   "default",
				ssl:      true,
			},
			want: "postgres://root:geheim@localhost:5432/default?sslmode=enable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dataSource(tt.args.host, tt.args.port, tt.args.user, tt.args.password, tt.args.dbname, tt.args.ssl); got != tt.want {
				t.Errorf("dataSource() = %v, want %v", got, tt.want)
			}
		})
	}
}
