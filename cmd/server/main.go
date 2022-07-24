package main

import (
	"log"
	"os"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/v"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {

	app := &cli.App{
		Name:    "api-server",
		Usage:   "cli for metal cloud",
		Version: v.V.String(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "grpc-server-endpoint",
				Value:   "localhost:9090",
				Usage:   "gRPC server endpoint",
				EnvVars: []string{"GOIPAM_GRPC_SERVER_ENDPOINT"},
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
				Usage:   "start memory backend",
				Action: func(ctx *cli.Context) error {
					c := getConfig(ctx)
					c.Ipamer = goipam.New()
					s := newServer(c)
					if err := s.Run(); err != nil {
						log.Fatal(err)
					}
					return nil
				},
			},
			{
				Name:    "postgres",
				Aliases: []string{"pg"},
				Usage:   "start postgres backend",
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
				},
				Action: func(ctx *cli.Context) error {
					c := getConfig(ctx)
					s := newServer(c)
					host := ctx.String("host")
					port := ctx.String("port")
					user := ctx.String("user")
					password := ctx.String("password")
					dbname := ctx.String("dbname")
					pgStorage, err := goipam.NewPostgresStorage(host, port, user, password, dbname, goipam.SSLModePrefer)
					if err != nil {
						return err
					}
					c.Ipamer = goipam.NewWithStorage(pgStorage)
					if err := s.Run(); err != nil {
						log.Fatal(err)
					}
					return nil
				},
			},
		},
		Action: func(ctx *cli.Context) error {
			c := getConfig(ctx)
			s := newServer(c)
			if err := s.Run(); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("Error in cli: %v", err)
	}

}

func getConfig(ctx *cli.Context) config {
	cfg := zap.NewProductionConfig()
	level, err := zap.ParseAtomicLevel(ctx.String("log-level"))
	if err != nil {
		panic(err)
	}
	cfg.Level = level
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zlog, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	return config{
		GrpcServerEndpoint: ctx.String("grpc-server-endpoint"),
		Log:                zlog.Sugar(),
	}
}
