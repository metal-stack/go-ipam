package ipam

type sql struct {
	tsql *tsql
}

type prefixJSON struct {
	*Prefix
	AvailableChildPrefixes map[string]bool // available child prefixes of this prefix
	ChildPrefixLength      int             // the length of the child prefixes
	IPs                    map[string]bool // The ips contained in this prefix
	Version                int64           // Version is used for optimistic locking
}

func (p *prefixJSON) toPrefix() *Prefix {
	return &Prefix{
		Cidr:                   p.Cidr,
		ParentCidr:             p.ParentCidr,
		availableChildPrefixes: p.AvailableChildPrefixes,
		childPrefixLength:      p.ChildPrefixLength,
		ips:                    p.IPs,
		version:                p.Version,
	}
}

func (p *Prefix) toPrefixJSON() *prefixJSON {
	return &prefixJSON{
		Prefix: &Prefix{
			Cidr:       p.Cidr,
			ParentCidr: p.ParentCidr,
		},
		AvailableChildPrefixes: p.availableChildPrefixes,
		ChildPrefixLength:      p.childPrefixLength,
		IPs:                    p.ips,
		Version:                p.version,
	}
}

func (s *sql) CreatePrefix(prefix *Prefix) (*Prefix, error) {
	err := s.tsql.Begin()
	if err != nil {
		return nil, err
	}
	prefix, err = s.tsql.CreatePrefix(prefix)
	if err != nil {
		return nil, s.tsql.Rollback()
	}
	return prefix, s.tsql.Commit()
}

func (s *sql) ReadPrefix(prefix string) (*Prefix, error) {
	return s.tsql.ReadPrefix(prefix)
}

func (s *sql) ReadAllPrefixes() ([]*Prefix, error) {
	return s.tsql.ReadAllPrefixes()
}

func (s *sql) UpdatePrefix(prefix *Prefix) (*Prefix, error) {
	err := s.tsql.Begin()
	if err != nil {
		return nil, err
	}
	prefix, err = s.tsql.UpdatePrefix(prefix)
	if err != nil {
		return nil, s.tsql.Rollback()
	}
	return prefix, s.tsql.Commit()
}

func (s *sql) DeletePrefix(prefix *Prefix) (*Prefix, error) {
	err := s.tsql.Begin()
	if err != nil {
		return nil, err
	}
	prefix, err = s.tsql.DeletePrefix(prefix)
	if err != nil {
		return nil, s.tsql.Rollback()
	}
	return prefix, s.tsql.Commit()

}
