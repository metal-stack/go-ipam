package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect"
	compress "github.com/klauspost/connect-compress/v2"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/v"
	"github.com/urfave/cli/v3"
)

func main() {

	app := &cli.Command{
		Name:    "cli",
		Usage:   "cli for go-ipam",
		Version: v.V.String(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "grpc-server-endpoint",
				Value:   "http://localhost:9090",
				Usage:   "gRPC server endpoint",
				Sources: cli.EnvVars("GOIPAM_CLI_GRPC_SERVER_ENDPOINT"),
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "prefix",
				Aliases: []string{"p"},
				Usage:   "prefix manipulation",
				Commands: []*cli.Command{
					{
						Name:  "create",
						Usage: "create a prefix",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "cidr",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c := client(cmd)
							result, err := c.CreatePrefix(context.Background(), connect.NewRequest(&v1.CreatePrefixRequest{
								Cidr: cmd.String("cidr"),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("prefix:%q created\n", result.Msg.GetPrefix().GetCidr())
							return nil
						},
					},
					{
						Name:  "acquire",
						Usage: "acquire a child prefix",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "parent",
							},
							&cli.UintFlag{
								Name: "length",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c := client(cmd)
							result, err := c.AcquireChildPrefix(context.Background(), connect.NewRequest(&v1.AcquireChildPrefixRequest{
								Cidr:   cmd.String("parent"),
								Length: uint32(cmd.Uint("length")), // nolint:gosec
							}))

							if err != nil {
								return err
							}
							fmt.Printf("child prefix:%q from %q created\n", result.Msg.GetPrefix().GetCidr(), result.Msg.GetPrefix().GetParentCidr())
							return nil
						},
					},
					{
						Name:  "release",
						Usage: "release a child prefix",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "cidr",
							},
						},
						Action: func(cctx context.Context, cmd *cli.Command) error {
							c := client(cmd)
							result, err := c.ReleaseChildPrefix(context.Background(), connect.NewRequest(&v1.ReleaseChildPrefixRequest{
								Cidr: cmd.String("cidr"),
							}))

							if err != nil {
								return err
							}
							if result.Msg == nil || result.Msg.GetPrefix() == nil {
								return fmt.Errorf("result contains no prefix")
							}
							fmt.Printf("child prefix:%q from %q released\n", result.Msg.GetPrefix().GetCidr(), result.Msg.GetPrefix().GetParentCidr())
							return nil
						},
					},
					{
						Name:  "list",
						Usage: "list all prefixes",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c := client(cmd)
							result, err := c.ListPrefixes(context.Background(), connect.NewRequest(&v1.ListPrefixesRequest{}))

							if err != nil {
								return err
							}
							for _, p := range result.Msg.GetPrefixes() {
								fmt.Printf("Prefix:%q parent:%q\n", p.GetCidr(), p.GetParentCidr())
							}
							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "delete a prefix",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "cidr",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c := client(cmd)
							result, err := c.DeletePrefix(context.Background(), connect.NewRequest(&v1.DeletePrefixRequest{
								Cidr: cmd.String("cidr"),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("prefix:%q deleted\n", result.Msg.GetPrefix().GetCidr())
							return nil
						},
					},
				},
			},
			{
				Name:    "ip",
				Aliases: []string{"i"},
				Usage:   "ip manipulation",
				Commands: []*cli.Command{
					{
						Name:  "acquire",
						Usage: "acquire a ip",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "prefix",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c := client(cmd)
							result, err := c.AcquireIP(context.Background(), connect.NewRequest(&v1.AcquireIPRequest{
								PrefixCidr: cmd.String("prefix"),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("ip:%q acquired\n", result.Msg.GetIp().GetIp())
							return nil
						},
					},
					{
						Name:  "release",
						Usage: "release a ip",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "ip",
							},
							&cli.StringFlag{
								Name: "prefix",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c := client(cmd)
							result, err := c.ReleaseIP(context.Background(), connect.NewRequest(&v1.ReleaseIPRequest{
								Ip:         cmd.String("ip"),
								PrefixCidr: cmd.String("prefix"),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("ip:%q released\n", result.Msg.GetIp().GetIp())
							return nil
						},
					},
				},
			},
			{
				Name:  "backup",
				Usage: "create and restore a backup",
				Commands: []*cli.Command{
					{
						Name:  "create",
						Usage: "create a json file of the whole ipam db for backup purpose",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c := client(cmd)
							result, err := c.Dump(context.Background(), connect.NewRequest(&v1.DumpRequest{}))
							if err != nil {
								return err
							}
							fmt.Println(result.Msg.GetDump())
							return nil
						},
					},
					{
						Name:  "restore",
						Usage: "load the whole ipam db from json file, previously created, only works if database is already empty",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "file",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c := client(cmd)
							json, err := os.ReadFile(cmd.String("file"))
							if err != nil {
								return err
							}
							_, err = c.Load(context.Background(), connect.NewRequest(&v1.LoadRequest{
								Dump: string(json),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("database restored\n")
							return nil
						},
					},
				},
			},
		},
	}
	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func client(cmd *cli.Command) apiv1connect.IpamServiceClient {

	return apiv1connect.NewIpamServiceClient(
		http.DefaultClient,
		cmd.String("grpc-server-endpoint"),
		connect.WithGRPC(),
		compress.WithAll(compress.LevelBalanced),
	)
}
