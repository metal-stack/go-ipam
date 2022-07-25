package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/bufbuild/connect-go"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/v"
	"github.com/urfave/cli/v2"
)

func main() {

	app := &cli.App{
		Name:    "cli",
		Usage:   "cli for go-ipam",
		Version: v.V.String(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "grpc-server-endpoint",
				Value:   "http://localhost:9090",
				Usage:   "gRPC server endpoint",
				EnvVars: []string{"GOIPAM_GRPC_SERVER_ENDPOINT"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "prefix",
				Aliases: []string{"p"},
				Usage:   "prefix manipulation",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "create a prefix",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "cidr",
							},
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.CreatePrefix(context.Background(), connect.NewRequest(&v1.CreatePrefixRequest{
								Cidr: ctx.String("cidr"),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("prefix:%q created\n", result.Msg.Prefix.Cidr)
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
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.AcquireChildPrefix(context.Background(), connect.NewRequest(&v1.AcquireChildPrefixRequest{
								Cidr:   ctx.String("parent"),
								Length: uint32(ctx.Uint("length")),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("child prefix:%q from %q created\n", result.Msg.Prefix.Cidr, result.Msg.Prefix.ParentCidr)
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
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.ReleaseChildPrefix(context.Background(), connect.NewRequest(&v1.ReleaseChildPrefixRequest{
								Cidr: ctx.String("cidr"),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("child prefix:%q from %q released\n", result.Msg.Prefix.Cidr, result.Msg.Prefix.ParentCidr)
							return nil
						},
					},
					{
						Name:  "list",
						Usage: "list all prefixes",
						Flags: []cli.Flag{
							&cli.StringFlag{
								// FIXME not implemented
								Name: "namespace",
							},
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.ListPrefixes(context.Background(), connect.NewRequest(&v1.ListPrefixesRequest{
								Namespace: ctx.String("namespace"),
							}))

							if err != nil {
								return err
							}
							for _, p := range result.Msg.Prefixes {
								fmt.Printf("Prefix:%q parent:%q\n", p.Cidr, p.ParentCidr)
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
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.DeletePrefix(context.Background(), connect.NewRequest(&v1.DeletePrefixRequest{
								Cidr: ctx.String("cidr"),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("prefix:%q deleted\n", result.Msg.Prefix.Cidr)
							return nil
						},
					},
				},
			},
			{
				Name:    "ip",
				Aliases: []string{"i"},
				Usage:   "ip manipulation",
				Subcommands: []*cli.Command{
					{
						Name:  "acquire",
						Usage: "acquire a ip",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "prefix",
							},
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.AcquireIP(context.Background(), connect.NewRequest(&v1.AcquireIPRequest{
								PrefixCidr: ctx.String("prefix"),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("ip:%q acquired\n", result.Msg.Ip.Ip)
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
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.ReleaseIP(context.Background(), connect.NewRequest(&v1.ReleaseIPRequest{
								Ip:         ctx.String("ip"),
								PrefixCidr: ctx.String("prefix"),
							}))

							if err != nil {
								return err
							}
							fmt.Printf("ip:%q released\n", result.Msg.Ip.Ip)
							return nil
						},
					},
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func client(ctx *cli.Context) apiv1connect.IpamServiceClient {
	return apiv1connect.NewIpamServiceClient(
		http.DefaultClient,
		ctx.String("grpc-server-endpoint"),
		connect.WithGRPC(),
	)
}
