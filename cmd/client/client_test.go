package main

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func newTestServer(t *testing.T) (string, *http.Client) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		service.New(log, goipam.New(t.Context())),
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	return server.URL, server.Client()
}

func runCLI(t *testing.T, endpoint string, httpClient *http.Client, args ...string) (string, error) {
	t.Helper()
	app := newApp()
	var out bytes.Buffer
	app.Writer = &out
	app.ErrWriter = &out
	app.Metadata = map[string]any{
		metadataHTTPClient: httpClient,
	}

	cmdArgs := append([]string{"cli", "--" + flagServerEndpoint, endpoint}, args...)
	err := app.Run(cmdArgs)
	return out.String(), err
}

func findCommand(t *testing.T, commands []*cli.Command, names ...string) *cli.Command {
	t.Helper()
	current := commands
	var found *cli.Command
	for _, name := range names {
		found = nil
		for _, cmd := range current {
			if cmd.Name == name || slices.Contains(cmd.Aliases, name) {
				found = cmd
				break
			}
		}
		require.NotNilf(t, found, "command %q not found", strings.Join(names, " "))
		current = found.Subcommands
	}
	return found
}

func commandFlagNames(cmd *cli.Command) []string {
	var names []string
	for _, f := range cmd.Flags {
		names = append(names, f.Names()...)
	}
	return names
}

func TestAppHasAllCommandsAndFlags(t *testing.T) {
	t.Parallel()

	app := newApp()

	var globalFlags []string
	for _, f := range app.Flags {
		globalFlags = append(globalFlags, f.Names()...)
	}
	assert.Contains(t, globalFlags, flagServerEndpoint)

	tests := []struct {
		path  []string
		flags []string
	}{
		{[]string{"prefix", "create"}, []string{flagCidr, flagNamespace}},
		{[]string{"prefix", "get"}, []string{flagCidr, flagNamespace}},
		{[]string{"prefix", "acquire"}, []string{flagParent, flagLength, flagChildCidr, flagNamespace}},
		{[]string{"prefix", "release"}, []string{flagCidr, flagNamespace}},
		{[]string{"prefix", "list"}, []string{flagNamespace}},
		{[]string{"prefix", "usage"}, []string{flagCidr, flagNamespace}},
		{[]string{"prefix", "delete"}, []string{flagCidr, flagNamespace}},
		{[]string{"ip", "acquire"}, []string{flagPrefix, flagIP, flagNamespace}},
		{[]string{"ip", "release"}, []string{flagIP, flagPrefix, flagNamespace}},
		{[]string{"ip", "list"}, []string{flagPrefix, flagNamespace}},
		{[]string{"namespace", "create"}, []string{flagNamespace}},
		{[]string{"namespace", "list"}, nil},
		{[]string{"namespace", "delete"}, []string{flagNamespace}},
		{[]string{"backup", "create"}, []string{flagNamespace}},
		{[]string{"backup", "restore"}, []string{flagFile, flagNamespace}},
		{[]string{"version"}, nil},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.path, "_"), func(t *testing.T) {
			t.Parallel()
			cmd := findCommand(t, app.Commands, tt.path...)
			require.NotNil(t, cmd.Action, "command should have an action")
			names := commandFlagNames(cmd)
			for _, flag := range tt.flags {
				assert.Contains(t, names, flag, "command %q should have flag %q", strings.Join(tt.path, " "), flag)
			}
		})
	}
}

func TestAppAliases(t *testing.T) {
	t.Parallel()

	app := newApp()

	assert.Equal(t, "prefix", findCommand(t, app.Commands, "p").Name)
	assert.Equal(t, "ip", findCommand(t, app.Commands, "i").Name)
	assert.Equal(t, "namespace", findCommand(t, app.Commands, "ns").Name)
}

func TestPrefixCommands(t *testing.T) {
	t.Parallel()
	endpoint, httpClient := newTestServer(t)

	out, err := runCLI(t, endpoint, httpClient, "prefix", "create", "--cidr", "192.168.0.0/16")
	require.NoError(t, err)
	assert.Contains(t, out, `prefix:"192.168.0.0/16" created`)

	out, err = runCLI(t, endpoint, httpClient, "prefix", "get", "--cidr", "192.168.0.0/16")
	require.NoError(t, err)
	assert.Contains(t, out, `prefix:"192.168.0.0/16" parent:""`)

	out, err = runCLI(t, endpoint, httpClient, "prefix", "list")
	require.NoError(t, err)
	assert.Contains(t, out, `Prefix:"192.168.0.0/16" parent:""`)

	out, err = runCLI(t, endpoint, httpClient, "prefix", "usage", "--cidr", "192.168.0.0/16")
	require.NoError(t, err)
	assert.Contains(t, out, "available ips:")
	assert.Contains(t, out, "acquired ips:")

	out, err = runCLI(t, endpoint, httpClient, "prefix", "acquire", "--parent", "192.168.0.0/16", "--child-cidr", "192.168.1.0/24")
	require.NoError(t, err)
	assert.Contains(t, out, `child prefix:"192.168.1.0/24" from "192.168.0.0/16" created`)

	out, err = runCLI(t, endpoint, httpClient, "prefix", "get", "--cidr", "192.168.1.0/24")
	require.NoError(t, err)
	assert.Contains(t, out, `prefix:"192.168.1.0/24" parent:"192.168.0.0/16"`)

	out, err = runCLI(t, endpoint, httpClient, "prefix", "release", "--cidr", "192.168.1.0/24")
	require.NoError(t, err)
	assert.Contains(t, out, `child prefix:"192.168.1.0/24" from "192.168.0.0/16" released`)

	out, err = runCLI(t, endpoint, httpClient, "prefix", "delete", "--cidr", "192.168.0.0/16")
	require.NoError(t, err)
	assert.Contains(t, out, `prefix:"192.168.0.0/16" deleted`)
}

func TestPrefixAcquireByLength(t *testing.T) {
	t.Parallel()
	endpoint, httpClient := newTestServer(t)

	_, err := runCLI(t, endpoint, httpClient, "prefix", "create", "--cidr", "10.10.0.0/16")
	require.NoError(t, err)

	out, err := runCLI(t, endpoint, httpClient, "prefix", "acquire", "--parent", "10.10.0.0/16", "--length", "24")
	require.NoError(t, err)
	assert.Contains(t, out, `child prefix:"10.10.0.0/24" from "10.10.0.0/16" created`)
}

func TestIPCommands(t *testing.T) {
	t.Parallel()
	endpoint, httpClient := newTestServer(t)

	_, err := runCLI(t, endpoint, httpClient, "prefix", "create", "--cidr", "10.0.0.0/24")
	require.NoError(t, err)

	out, err := runCLI(t, endpoint, httpClient, "ip", "acquire", "--prefix", "10.0.0.0/24", "--ip", "10.0.0.5")
	require.NoError(t, err)
	assert.Contains(t, out, `ip:"10.0.0.5" acquired`)

	out, err = runCLI(t, endpoint, httpClient, "ip", "list", "--prefix", "10.0.0.0/24")
	require.NoError(t, err)
	assert.Contains(t, out, `ip:"10.0.0.5"`)

	out, err = runCLI(t, endpoint, httpClient, "ip", "acquire", "--prefix", "10.0.0.0/24")
	require.NoError(t, err)
	assert.Contains(t, out, `ip:"10.0.0.1" acquired`)

	out, err = runCLI(t, endpoint, httpClient, "ip", "release", "--prefix", "10.0.0.0/24", "--ip", "10.0.0.5")
	require.NoError(t, err)
	assert.Contains(t, out, `ip:"10.0.0.5" released`)

	out, err = runCLI(t, endpoint, httpClient, "ip", "list", "--prefix", "10.0.0.0/24")
	require.NoError(t, err)
	assert.NotContains(t, out, `ip:"10.0.0.5"`)
}

func TestNamespaceCommands(t *testing.T) {
	t.Parallel()
	endpoint, httpClient := newTestServer(t)

	out, err := runCLI(t, endpoint, httpClient, "namespace", "create", "--namespace", "tenant-a")
	require.NoError(t, err)
	assert.Contains(t, out, `namespace:"tenant-a" created`)

	out, err = runCLI(t, endpoint, httpClient, "namespace", "list")
	require.NoError(t, err)
	assert.Contains(t, out, `namespace:"tenant-a"`)

	out, err = runCLI(t, endpoint, httpClient, "prefix", "create", "--namespace", "tenant-a", "--cidr", "172.16.0.0/24")
	require.NoError(t, err)
	assert.Contains(t, out, `prefix:"172.16.0.0/24" created`)

	out, err = runCLI(t, endpoint, httpClient, "prefix", "list", "--namespace", "tenant-a")
	require.NoError(t, err)
	assert.Contains(t, out, `Prefix:"172.16.0.0/24"`)

	out, err = runCLI(t, endpoint, httpClient, "prefix", "list")
	require.NoError(t, err)
	assert.NotContains(t, out, `Prefix:"172.16.0.0/24"`)

	_, err = runCLI(t, endpoint, httpClient, "namespace", "delete", "--namespace", "tenant-a")
	require.Error(t, err, "a namespace with allocated prefixes must not be deletable")

	out, err = runCLI(t, endpoint, httpClient, "prefix", "delete", "--namespace", "tenant-a", "--cidr", "172.16.0.0/24")
	require.NoError(t, err)
	assert.Contains(t, out, `prefix:"172.16.0.0/24" deleted`)

	out, err = runCLI(t, endpoint, httpClient, "namespace", "delete", "--namespace", "tenant-a")
	require.NoError(t, err)
	assert.Contains(t, out, `namespace:"tenant-a" deleted`)

	out, err = runCLI(t, endpoint, httpClient, "namespace", "list")
	require.NoError(t, err)
	assert.NotContains(t, out, `namespace:"tenant-a"`)
}

func TestNamespaceScopedIPCommands(t *testing.T) {
	t.Parallel()
	endpoint, httpClient := newTestServer(t)

	_, err := runCLI(t, endpoint, httpClient, "namespace", "create", "--namespace", "tenant-a")
	require.NoError(t, err)
	_, err = runCLI(t, endpoint, httpClient, "prefix", "create", "--namespace", "tenant-a", "--cidr", "172.17.0.0/24")
	require.NoError(t, err)

	out, err := runCLI(t, endpoint, httpClient, "ip", "acquire", "--namespace", "tenant-a", "--prefix", "172.17.0.0/24")
	require.NoError(t, err)
	assert.Contains(t, out, `ip:"172.17.0.1" acquired`)

	out, err = runCLI(t, endpoint, httpClient, "ip", "list", "--namespace", "tenant-a", "--prefix", "172.17.0.0/24")
	require.NoError(t, err)
	assert.Contains(t, out, `ip:"172.17.0.1"`)

	out, err = runCLI(t, endpoint, httpClient, "ip", "release", "--namespace", "tenant-a", "--prefix", "172.17.0.0/24", "--ip", "172.17.0.1")
	require.NoError(t, err)
	assert.Contains(t, out, `ip:"172.17.0.1" released`)
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()
	endpoint, httpClient := newTestServer(t)

	out, err := runCLI(t, endpoint, httpClient, "version")
	require.NoError(t, err)
	assert.Contains(t, out, "version:")
	assert.Contains(t, out, "revision:")
	assert.Contains(t, out, "gitsha1:")
	assert.Contains(t, out, "builddate:")
}

func TestBackupCommands(t *testing.T) {
	t.Parallel()

	sourceEndpoint, sourceClient := newTestServer(t)
	_, err := runCLI(t, sourceEndpoint, sourceClient, "prefix", "create", "--cidr", "192.168.10.0/24")
	require.NoError(t, err)

	dump, err := runCLI(t, sourceEndpoint, sourceClient, "backup", "create")
	require.NoError(t, err)
	require.NotEmpty(t, dump)

	dumpFile := filepath.Join(t.TempDir(), "dump.json")
	require.NoError(t, os.WriteFile(dumpFile, []byte(dump), 0o600))

	targetEndpoint, targetClient := newTestServer(t)
	out, err := runCLI(t, targetEndpoint, targetClient, "backup", "restore", "--file", dumpFile)
	require.NoError(t, err)
	assert.Contains(t, out, "database restored")

	out, err = runCLI(t, targetEndpoint, targetClient, "prefix", "list")
	require.NoError(t, err)
	assert.Contains(t, out, `Prefix:"192.168.10.0/24"`)
}

func TestBackupRestoreMissingFile(t *testing.T) {
	t.Parallel()
	endpoint, httpClient := newTestServer(t)

	_, err := runCLI(t, endpoint, httpClient, "backup", "restore", "--file", filepath.Join(t.TempDir(), "does-not-exist.json"))
	require.Error(t, err)
}

func TestCommandErrorsAreReturned(t *testing.T) {
	t.Parallel()
	endpoint, httpClient := newTestServer(t)

	_, err := runCLI(t, endpoint, httpClient, "prefix", "get", "--cidr", "203.0.113.0/24")
	require.Error(t, err)

	_, err = runCLI(t, endpoint, httpClient, "prefix", "delete", "--cidr", "203.0.113.0/24")
	require.Error(t, err)
}
