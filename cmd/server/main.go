package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/v"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	app := &cli.App{
		Name:    "go-ipam server",
		Usage:   "grpc server for go ipam",
		Version: v.V.String(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "grpc-server-endpoint",
				Value:   "localhost:9090",
				Usage:   "gRPC server endpoint",
				EnvVars: []string{"GOIPAM_GRPC_SERVER_ENDPOINT"},
			},
			&cli.StringFlag{
				Name:    "metrics-endpoint",
				Value:   "localhost:2112",
				Usage:   "metrics endpoint",
				EnvVars: []string{"GOIPAM_METRICS_ENDPOINT"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "log-level can be one of error|warn|info|debug",
				EnvVars: []string{"GOIPAM_LOG_LEVEL"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "memory",
				Aliases: []string{"m"},
				Usage:   "start with memory backend",
				Action: func(ctx *cli.Context) error {
					c := getConfig(ctx)
					c.Storage = goipam.NewMemory(ctx.Context)
					s := newServer(c)
					return s.Run()
				},
			},
			{
				Name:    "postgres",
				Aliases: []string{"pg"},
				Usage:   "start with postgres backend",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "host",
						Value:   "localhost",
						Usage:   "postgres db hostname",
						EnvVars: []string{"GOIPAM_PG_HOST"},
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "5432",
						Usage:   "postgres db port",
						EnvVars: []string{"GOIPAM_PG_PORT"},
					},
					&cli.StringFlag{
						Name:    "user",
						Value:   "go-ipam",
						Usage:   "postgres db user",
						EnvVars: []string{"GOIPAM_PG_USER"},
					},
					&cli.StringFlag{
						Name:    "password",
						Value:   "secret",
						Usage:   "postgres db password",
						EnvVars: []string{"GOIPAM_PG_PASSWORD"},
					},
					&cli.StringFlag{
						Name:    "dbname",
						Value:   "goipam",
						Usage:   "postgres db name",
						EnvVars: []string{"GOIPAM_PG_DBNAME"},
					},
					&cli.StringFlag{
						Name:    "sslmode",
						Value:   "disable",
						Usage:   "postgres sslmode, possible values: disable|require|verify-ca|verify-full",
						EnvVars: []string{"GOIPAM_PG_SSLMODE"},
					},
				},
				Action: func(ctx *cli.Context) error {
					c := getConfig(ctx)
					host := ctx.String("host")
					port := ctx.String("port")
					user := ctx.String("user")
					password := ctx.String("password")
					dbname := ctx.String("dbname")
					sslmode := ctx.String("sslmode")
					pgStorage, err := goipam.NewPostgresStorage(host, port, user, password, dbname, goipam.SSLMode(sslmode))
					if err != nil {
						return err
					}
					c.Storage = pgStorage
					s := newServer(c)
					return s.Run()
				},
			},
			{
				Name:  "redis",
				Usage: "start with redis backend",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "host",
						Value:   "localhost",
						Usage:   "redis db hostname",
						EnvVars: []string{"GOIPAM_REDIS_HOST"},
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "6379",
						Usage:   "redis db port",
						EnvVars: []string{"GOIPAM_REDIS_PORT"},
					},
				},
				Action: func(ctx *cli.Context) error {
					c := getConfig(ctx)
					host := ctx.String("host")
					port := ctx.String("port")
					var err error
					c.Storage, err = goipam.NewRedis(ctx.Context, host, port)
					if err != nil {
						return err
					}

					s := newServer(c)
					return s.Run()
				},
			},
			{
				Name:  "etcd",
				Usage: "start with etcd backend",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "host",
						Value:   "localhost",
						Usage:   "etcd db hostname",
						EnvVars: []string{"GOIPAM_ETCD_HOST"},
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "2379",
						Usage:   "etcd db port",
						EnvVars: []string{"GOIPAM_ETCD_PORT"},
					},
					&cli.StringFlag{
						Name:    "cert-file",
						Value:   "cert.pem",
						Usage:   "etcd cert file",
						EnvVars: []string{"GOIPAM_ETCD_CERT_FILE"},
					},
					&cli.StringFlag{
						Name:    "key-file",
						Value:   "key.pem",
						Usage:   "etcd key file",
						EnvVars: []string{"GOIPAM_ETCD_KEY_FILE"},
					},
					&cli.BoolFlag{
						Name:    "insecure-skip-verify",
						Value:   false,
						Usage:   "skip tls certification verification",
						EnvVars: []string{"GOIPAM_ETCD_INSECURE_SKIP_VERIFY"},
					},
				},
				Action: func(ctx *cli.Context) error {
					c := getConfig(ctx)
					host := ctx.String("host")
					port := ctx.String("port")
					certFile := ctx.String("cert-file")
					keyFile := ctx.String("key-file")
					cert, err := os.ReadFile(certFile)
					if err != nil {
						return err
					}
					key, err := os.ReadFile(keyFile)
					if err != nil {
						return err
					}
					insecureSkip := ctx.Bool("insecure-skip-verify")

					c.Storage, err = goipam.NewEtcd(ctx.Context, host, port, cert, key, insecureSkip)
					if err != nil {
						return err
					}
					s := newServer(c)
					return s.Run()
				},
			},
			{
				Name:  "mongodb",
				Usage: "start with mongodb backend",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "host",
						Value:   "localhost",
						Usage:   "mongodb db hostname",
						EnvVars: []string{"GOIPAM_MONGODB_HOST"},
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "27017",
						Usage:   "mongodb db port",
						EnvVars: []string{"GOIPAM_MONGODB_PORT"},
					},
					&cli.StringFlag{
						Name:    "db-name",
						Value:   "go-ipam",
						Usage:   "mongodb db name",
						EnvVars: []string{"GOIPAM_MONGODB_DB_NAME"},
					},
					&cli.StringFlag{
						Name:    "collection-name",
						Value:   "prefixes",
						Usage:   "mongodb db collection name",
						EnvVars: []string{"GOIPAM_MONGODB_COLLECTION_NAME"},
					},
					&cli.StringFlag{
						Name:    "user",
						Value:   "mongodb",
						Usage:   "mongodb db user",
						EnvVars: []string{"GOIPAM_MONGODB_USER"},
					},
					&cli.StringFlag{
						Name:    "password",
						Value:   "mongodb",
						Usage:   "mongodb db password",
						EnvVars: []string{"GOIPAM_MONGODB_PASSWORD"},
					},
				},
				Action: func(ctx *cli.Context) error {
					c := getConfig(ctx)
					host := ctx.String("host")
					port := ctx.String("port")
					user := ctx.String("user")
					password := ctx.String("password")
					dbname := ctx.String("db-name")

					opts := options.Client()
					opts.ApplyURI(fmt.Sprintf(`mongodb://%s:%s`, host, port))
					opts.Auth = &options.Credential{
						AuthMechanism: `SCRAM-SHA-1`,
						Username:      user,
						Password:      password,
					}

					mongocfg := goipam.MongoConfig{
						DatabaseName:       dbname,
						MongoClientOptions: opts,
					}
					db, err := goipam.NewMongo(context.Background(), mongocfg)
					if err != nil {
						return err
					}
					c.Storage = db

					s := newServer(c)
					return s.Run()
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("Error in cli: %v", err)
	}

}

func getConfig(ctx *cli.Context) config {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	switch ctx.String("loglevel") {
	case "debug":
		opts.Level = slog.LevelDebug
	case "error":
		opts.Level = slog.LevelError
	}

	return config{
		GrpcServerEndpoint: ctx.String("grpc-server-endpoint"),
		MetricsEndpoint:    ctx.String("metrics-endpoint"),
		Log:                slog.New(slog.NewJSONHandler(os.Stdout, opts)),
	}
}
