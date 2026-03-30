package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/v"
	"github.com/urfave/cli/v3"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	app := &cli.Command{
		Name:    "go-ipam server",
		Usage:   "grpc server for go ipam",
		Version: v.V.String(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "grpc-server-endpoint",
				Value:   ":9090",
				Usage:   "gRPC server endpoint",
				Sources: cli.EnvVars("GOIPAM_GRPC_SERVER_ENDPOINT"),
				Local:   true,
			},
			&cli.StringFlag{
				Name:    "metrics-endpoint",
				Value:   ":2112",
				Usage:   "metrics endpoint",
				Sources: cli.EnvVars("GOIPAM_METRICS_ENDPOINT"),
				Local:   true,
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "log-level can be one of error|warn|info|debug",
				Sources: cli.EnvVars("GOIPAM_LOG_LEVEL"),
				Local:   true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "memory",
				Aliases: []string{"m"},
				Usage:   "start with memory backend",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					c := getConfig(cmd)
					c.Storage = goipam.NewMemory(ctx)
					s := newServer(c)
					return s.Run()
				},
			},
			{
				Name:    "file",
				Aliases: []string{"f", "local"},
				Usage:   "start with local JSON file backend",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "path",
						Value:       goipam.DefaultLocalFilePath,
						DefaultText: "~/.local/share/go-ipam/ipam-db.json",
						Usage:       "path to the file",
						Sources:     cli.EnvVars("GOIPAM_FILE_PATH"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					c := getConfig(cmd)
					c.Storage = goipam.NewLocalFile(ctx, cmd.String("path"))
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
						Sources: cli.EnvVars("GOIPAM_PG_HOST"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "5432",
						Usage:   "postgres db port",
						Sources: cli.EnvVars("GOIPAM_PG_PORT"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "user",
						Value:   "go-ipam",
						Usage:   "postgres db user",
						Sources: cli.EnvVars("GOIPAM_PG_USER"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "password",
						Value:   "secret",
						Usage:   "postgres db password",
						Sources: cli.EnvVars("GOIPAM_PG_PASSWORD"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "dbname",
						Value:   "goipam",
						Usage:   "postgres db name",
						Sources: cli.EnvVars("GOIPAM_PG_DBNAME"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "sslmode",
						Value:   "disable",
						Usage:   "postgres sslmode, possible values: disable|require|verify-ca|verify-full",
						Sources: cli.EnvVars("GOIPAM_PG_SSLMODE"),
						Local:   true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					c := getConfig(cmd)
					host := cmd.String("host")
					port := cmd.String("port")
					user := cmd.String("user")
					password := cmd.String("password")
					dbname := cmd.String("dbname")
					sslmode := cmd.String("sslmode")
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
						Sources: cli.EnvVars("GOIPAM_REDIS_HOST"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "6379",
						Usage:   "redis db port",
						Sources: cli.EnvVars("GOIPAM_REDIS_PORT"),
						Local:   true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					c := getConfig(cmd)
					host := cmd.String("host")
					port := cmd.String("port")
					var err error
					c.Storage, err = goipam.NewRedis(ctx, host, port)
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
						Sources: cli.EnvVars("GOIPAM_ETCD_HOST"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "2379",
						Usage:   "etcd db port",
						Sources: cli.EnvVars("GOIPAM_ETCD_PORT"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "cert-file",
						Value:   "cert.pem",
						Usage:   "etcd cert file",
						Sources: cli.EnvVars("GOIPAM_ETCD_CERT_FILE"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "key-file",
						Value:   "key.pem",
						Usage:   "etcd key file",
						Sources: cli.EnvVars("GOIPAM_ETCD_KEY_FILE"),
						Local:   true,
					},
					&cli.BoolFlag{
						Name:    "insecure-skip-verify",
						Value:   false,
						Usage:   "skip tls certification verification",
						Sources: cli.EnvVars("GOIPAM_ETCD_INSECURE_SKIP_VERIFY"),
						Local:   true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					c := getConfig(cmd)
					host := cmd.String("host")
					port := cmd.String("port")
					certFile := cmd.String("cert-file")
					keyFile := cmd.String("key-file")
					cert, err := os.ReadFile(certFile)
					if err != nil {
						return err
					}
					key, err := os.ReadFile(keyFile)
					if err != nil {
						return err
					}
					insecureSkip := cmd.Bool("insecure-skip-verify")

					c.Storage, err = goipam.NewEtcd(ctx, host, port, cert, key, insecureSkip)
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
						Sources: cli.EnvVars("GOIPAM_MONGODB_HOST"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "port",
						Value:   "27017",
						Usage:   "mongodb db port",
						Sources: cli.EnvVars("GOIPAM_MONGODB_PORT"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "db-name",
						Value:   "go-ipam",
						Usage:   "mongodb db name",
						Sources: cli.EnvVars("GOIPAM_MONGODB_DB_NAME"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "collection-name",
						Value:   "prefixes",
						Usage:   "mongodb db collection name",
						Sources: cli.EnvVars("GOIPAM_MONGODB_COLLECTION_NAME"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "user",
						Value:   "mongodb",
						Usage:   "mongodb db user",
						Sources: cli.EnvVars("GOIPAM_MONGODB_USER"),
						Local:   true,
					},
					&cli.StringFlag{
						Name:    "password",
						Value:   "mongodb",
						Usage:   "mongodb db password",
						Sources: cli.EnvVars("GOIPAM_MONGODB_PASSWORD"),
						Local:   true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					c := getConfig(cmd)
					host := cmd.String("host")
					port := cmd.String("port")
					user := cmd.String("user")
					password := cmd.String("password")
					dbname := cmd.String("db-name")

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

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatalf("unable to start ipam service: %v", err)
	}
}

func getConfig(cmd *cli.Command) config {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	switch cmd.String("log-level") {
	case "debug":
		opts.Level = slog.LevelDebug
	case "error":
		opts.Level = slog.LevelError
	}

	return config{
		GrpcServerEndpoint: cmd.String("grpc-server-endpoint"),
		MetricsEndpoint:    cmd.String("metrics-endpoint"),
		Log:                slog.New(slog.NewJSONHandler(os.Stdout, opts)),
	}
}
