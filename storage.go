package ipam

// Storage is a interface to store ipam objects.
type Storage interface {
	CreatePrefix(prefix Prefix) (Prefix, error)
	ReadPrefix(prefix string) (Prefix, error)
	ReadAllPrefixes() ([]Prefix, error)
	UpdatePrefix(prefix Prefix) (Prefix, error)
	DeletePrefix(prefix Prefix) (Prefix, error)
}

// OptimisticLockError indicates that the operation could not be executed because the dataset to update has changed in the meantime.
// clients can decide to read the current dataset and retry the operation.
type OptimisticLockError struct {
	msg string
}

func (o OptimisticLockError) Error() string {
	return o.msg
}

func NewOptimisticLockError(msg string) OptimisticLockError {
	return OptimisticLockError{msg: msg}
}
