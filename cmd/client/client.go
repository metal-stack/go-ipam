package main

import (
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect"
	compress "github.com/klauspost/connect-compress/v2"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/v"
	"github.com/urfave/cli/v2"
)

const (
	metadataHTTPClient = "http-client"

	flagCidr           = "cidr"
	flagNamespace      = "namespace"
	flagParent         = "parent"
	flagPrefix         = "prefix"
	flagLength         = "length"
	flagChildCidr      = "child-cidr"
	flagIP             = "ip"
	flagFile           = "file"
	flagServerEndpoint = "grpc-server-endpoint"
)

func main() {
	err := newApp().Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func newApp() *cli.App {
	return &cli.App{
		Name:    "cli",
		Usage:   "cli for go-ipam",
		Version: v.V.String(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    flagServerEndpoint,
				Value:   "http://localhost:9090",
				Usage:   "gRPC server endpoint",
				EnvVars: []string{"GOIPAM_CLI_GRPC_SERVER_ENDPOINT"},
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
							cidrFlag("cidr of the prefix to create"),
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.CreatePrefix(ctx.Context, &v1.CreatePrefixRequest{
								Cidr:      ctx.String(flagCidr),
								Namespace: namespace(ctx),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "prefix:%q created\n", result.GetPrefix().GetCidr())
							return nil
						},
					},
					{
						Name:  "get",
						Usage: "get a prefix",
						Flags: []cli.Flag{
							cidrFlag("cidr of the prefix to get"),
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.GetPrefix(ctx.Context, &v1.GetPrefixRequest{
								Cidr:      ctx.String(flagCidr),
								Namespace: namespace(ctx),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "prefix:%q parent:%q\n", result.GetPrefix().GetCidr(), result.GetPrefix().GetParentCidr())
							return nil
						},
					},
					{
						Name:  "acquire",
						Usage: "acquire a child prefix",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  flagParent,
								Usage: "parent prefix to acquire the child prefix from",
							},
							&cli.UintFlag{
								Name:  flagLength,
								Usage: "length of the child prefix to acquire",
							},
							&cli.StringFlag{
								Name:  flagChildCidr,
								Usage: "specific child cidr to acquire instead of a length",
							},
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.AcquireChildPrefix(ctx.Context, &v1.AcquireChildPrefixRequest{
								Cidr:      ctx.String(flagParent),
								Length:    uint32(ctx.Uint(flagLength)), // nolint:gosec
								ChildCidr: optionalString(ctx, flagChildCidr),
								Namespace: namespace(ctx),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "child prefix:%q from %q created\n", result.GetPrefix().GetCidr(), result.GetPrefix().GetParentCidr())
							return nil
						},
					},
					{
						Name:  "release",
						Usage: "release a child prefix",
						Flags: []cli.Flag{
							cidrFlag("cidr of the child prefix to release"),
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.ReleaseChildPrefix(ctx.Context, &v1.ReleaseChildPrefixRequest{
								Cidr:      ctx.String(flagCidr),
								Namespace: namespace(ctx),
							})

							if err != nil {
								return err
							}
							if result == nil || result.GetPrefix() == nil {
								return fmt.Errorf("result contains no prefix")
							}
							printOut(ctx, "child prefix:%q from %q released\n", result.GetPrefix().GetCidr(), result.GetPrefix().GetParentCidr())
							return nil
						},
					},
					{
						Name:  "list",
						Usage: "list all prefixes",
						Flags: []cli.Flag{
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.ListPrefixes(ctx.Context, &v1.ListPrefixesRequest{
								Namespace: namespace(ctx),
							})

							if err != nil {
								return err
							}
							for _, p := range result.GetPrefixes() {
								printOut(ctx, "Prefix:%q parent:%q\n", p.GetCidr(), p.GetParentCidr())
							}
							return nil
						},
					},
					{
						Name:  "usage",
						Usage: "get the usage of a prefix",
						Flags: []cli.Flag{
							cidrFlag("cidr of the prefix to get the usage for"),
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.PrefixUsage(ctx.Context, &v1.PrefixUsageRequest{
								Cidr:      ctx.String(flagCidr),
								Namespace: namespace(ctx),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "prefix:%q available ips:%d acquired ips:%d available smallest prefixes:%d acquired prefixes:%d\n",
								ctx.String(flagCidr),
								result.GetAvailableIps(),
								result.GetAcquiredIps(),
								result.GetAvailableSmallestPrefixes(),
								result.GetAcquiredPrefixes(),
							)
							for _, p := range result.GetAvailablePrefixes() {
								printOut(ctx, "available prefix:%q\n", p)
							}
							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "delete a prefix",
						Flags: []cli.Flag{
							cidrFlag("cidr of the prefix to delete"),
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.DeletePrefix(ctx.Context, &v1.DeletePrefixRequest{
								Cidr:      ctx.String(flagCidr),
								Namespace: namespace(ctx),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "prefix:%q deleted\n", result.GetPrefix().GetCidr())
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
								Name:  flagPrefix,
								Usage: "prefix to acquire the ip from",
							},
							&cli.StringFlag{
								Name:  flagIP,
								Usage: "specific ip to acquire instead of the next available one",
							},
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.AcquireIP(ctx.Context, &v1.AcquireIPRequest{
								PrefixCidr: ctx.String(flagPrefix),
								Ip:         optionalString(ctx, flagIP),
								Namespace:  namespace(ctx),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "ip:%q acquired\n", result.GetIp().GetIp())
							return nil
						},
					},
					{
						Name:  "release",
						Usage: "release a ip",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  flagIP,
								Usage: "ip to release",
							},
							&cli.StringFlag{
								Name:  flagPrefix,
								Usage: "prefix the ip belongs to",
							},
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.ReleaseIP(ctx.Context, &v1.ReleaseIPRequest{
								Ip:         ctx.String(flagIP),
								PrefixCidr: ctx.String(flagPrefix),
								Namespace:  namespace(ctx),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "ip:%q released\n", result.GetIp().GetIp())
							return nil
						},
					},
					{
						Name:  "list",
						Usage: "list all ips of a prefix",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  flagPrefix,
								Usage: "prefix to list the ips for",
							},
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.ListIPs(ctx.Context, &v1.ListIPsRequest{
								PrefixCidr: ctx.String(flagPrefix),
								Namespace:  namespace(ctx),
							})

							if err != nil {
								return err
							}
							for _, ip := range result.GetIp() {
								printOut(ctx, "ip:%q parent:%q\n", ip, result.GetParentPrefix())
							}
							return nil
						},
					},
				},
			},
			{
				Name:    "namespace",
				Aliases: []string{"ns"},
				Usage:   "namespace manipulation",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "create a namespace",
						Flags: []cli.Flag{
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							_, err := c.CreateNamespace(ctx.Context, &v1.CreateNamespaceRequest{
								Namespace: ctx.String(flagNamespace),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "namespace:%q created\n", ctx.String(flagNamespace))
							return nil
						},
					},
					{
						Name:  "list",
						Usage: "list all namespaces",
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.ListNamespaces(ctx.Context, &v1.ListNamespacesRequest{})

							if err != nil {
								return err
							}
							for _, n := range result.GetNamespace() {
								printOut(ctx, "namespace:%q\n", n)
							}
							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "delete a namespace",
						Flags: []cli.Flag{
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							_, err := c.DeleteNamespace(ctx.Context, &v1.DeleteNamespaceRequest{
								Namespace: ctx.String(flagNamespace),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "namespace:%q deleted\n", ctx.String(flagNamespace))
							return nil
						},
					},
				},
			},
			{
				Name:  "backup",
				Usage: "create and restore a backup",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "create a json file of the whole ipam db for backup purpose",
						Flags: []cli.Flag{
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							result, err := c.Dump(ctx.Context, &v1.DumpRequest{
								Namespace: namespace(ctx),
							})
							if err != nil {
								return err
							}
							printOut(ctx, "%s\n", result.GetDump())
							return nil
						},
					},
					{
						Name:  "restore",
						Usage: "load the whole ipam db from json file, previously created, only works if database is already empty",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  flagFile,
								Usage: "file containing the json dump to restore",
							},
							namespaceFlag(),
						},
						Action: func(ctx *cli.Context) error {
							c := client(ctx)
							json, err := os.ReadFile(ctx.String(flagFile))
							if err != nil {
								return err
							}
							_, err = c.Load(ctx.Context, &v1.LoadRequest{
								Dump:      string(json),
								Namespace: namespace(ctx),
							})

							if err != nil {
								return err
							}
							printOut(ctx, "database restored\n")
							return nil
						},
					},
				},
			},
			{
				Name:  "version",
				Usage: "print the version of the server",
				Action: func(ctx *cli.Context) error {
					c := client(ctx)
					result, err := c.Version(ctx.Context, &v1.VersionRequest{})
					if err != nil {
						return err
					}
					printOut(ctx, "version:%q revision:%q gitsha1:%q builddate:%q\n",
						result.GetVersion(),
						result.GetRevision(),
						result.GetGitSha1(),
						result.GetBuildDate(),
					)
					return nil
				},
			},
		},
	}
}

func printOut(ctx *cli.Context, format string, args ...any) {
	_, _ = fmt.Fprintf(ctx.App.Writer, format, args...)
}

func cidrFlag(usage string) cli.Flag {
	return &cli.StringFlag{
		Name:  flagCidr,
		Usage: usage,
	}
}

func namespaceFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    flagNamespace,
		Usage:   "namespace to operate in",
		EnvVars: []string{"GOIPAM_CLI_NAMESPACE"},
	}
}

// namespace returns the namespace if it was set, nil otherwise. This keeps the
// optional namespace semantics of the underlying protobuf messages intact.
func namespace(ctx *cli.Context) *string {
	if !ctx.IsSet(flagNamespace) {
		return nil
	}
	value := ctx.String(flagNamespace)
	if value == "" {
		return nil
	}
	return &value
}

// optionalString returns a pointer to the flag value, or nil if it is unset.
// It is used for optional string fields that must not be sent when empty.
func optionalString(ctx *cli.Context, flag string) *string {
	if !ctx.IsSet(flag) {
		return nil
	}
	value := ctx.String(flag)
	if value == "" {
		return nil
	}
	return &value
}

func client(ctx *cli.Context) apiv1connect.IpamServiceClient {
	httpClient := http.DefaultClient
	if c, ok := ctx.App.Metadata[metadataHTTPClient].(*http.Client); ok && c != nil {
		httpClient = c
	}

	return apiv1connect.NewIpamServiceClient(
		httpClient,
		ctx.String(flagServerEndpoint),
		connect.WithGRPC(),
		compress.WithAll(compress.LevelBalanced),
	)
}
