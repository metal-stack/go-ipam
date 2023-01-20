package ipam

import "context"

// Storage is a interface to store ipam objects.
type Storage interface {
	Name() string
	CreatePrefix(ctx context.Context, prefix Prefix) (Prefix, error)
	ReadPrefix(ctx context.Context, prefix string, namespace string) (Prefix, error)
	ReadPrefixes(ctx context.Context, namespace string) (Prefixes, error)
	DeleteAllPrefixes(ctx context.Context) error
	ReadAllPrefixes(ctx context.Context) (Prefixes, error)
	ReadAllPrefixCidrs(ctx context.Context, namespace string) ([]string, error)
	UpdatePrefix(ctx context.Context, prefix Prefix) (Prefix, error)
	DeletePrefix(ctx context.Context, prefix Prefix) (Prefix, error)
}
