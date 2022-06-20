package ipam

// Storage is a interface to store ipam objects.
type Storage interface {
	CreatePrefix(prefix Prefix) (Prefix, error)
	ReadPrefix(prefix string) (Prefix, error)
	DeleteAllPrefixes() error
	ReadAllPrefixes() (Prefixes, error)
	ReadAllPrefixCidrs() ([]string, error)
	UpdatePrefix(prefix Prefix) (Prefix, error)
	DeletePrefix(prefix Prefix) (Prefix, error)
}
