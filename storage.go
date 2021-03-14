package ipam

// Storage is a interface to store ipam objects.
type Storage interface {
	CreatePrefix(prefix Prefix) (Prefix, error)
	ReadPrefix(prefix, namespace string) (Prefix, error)
	ReadAllPrefixes() ([]Prefix, error)
	ReadAllPrefixCidrs() ([]string, error)
	UpdatePrefix(prefix Prefix) (Prefix, error)
	DeletePrefix(prefix Prefix) (Prefix, error)
}
