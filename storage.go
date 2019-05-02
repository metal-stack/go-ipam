package ipam

// Storage is a interface to store network objects.
type Storage interface {
	PrefixStorage
}

// PrefixStorage will do CRUD operations on a Prefix
type PrefixStorage interface {
	CreatePrefix(prefix *Prefix) (*Prefix, error)
	ReadPrefix(prefix string) (*Prefix, error)
	ReadAllPrefixes() ([]*Prefix, error)
	UpdatePrefix(prefix *Prefix) (*Prefix, error)
	DeletePrefix(prefix *Prefix) (*Prefix, error)
}
