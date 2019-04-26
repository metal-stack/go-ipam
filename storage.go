package ipam

// Storage is a interface to store network objects.
type Storage interface {
	NetworkStorage
	PrefixStorage
}

// NetworkStorage will do CRUD operations on a Network
type NetworkStorage interface {
	CreateNetwork(network *Network) (*Network, error)
	ReadNetwork(id string) (*Network, error)
	ReadAllNetworks() ([]*Network, error)
	UpdateNetwork(network *Network) (*Network, error)
	DeleteNetwork(network *Network) (*Network, error)
}

// PrefixStorage will do CRUD operations on a Prefix
type PrefixStorage interface {
	CreatePrefix(prefix *Prefix) (*Prefix, error)
	ReadPrefix(prefix string) (*Prefix, error)
	ReadAllPrefixes() ([]*Prefix, error)
	UpdatePrefix(prefix *Prefix) (*Prefix, error)
	DeletePrefix(prefix *Prefix) (*Prefix, error)
}
