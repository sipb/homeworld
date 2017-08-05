package authorities

type DelegatedAuthority struct {
	Name string
}

func NewDelegatedAuthority(name string) Authority {
	return &DelegatedAuthority{name}
}

func (d *DelegatedAuthority) GetPublicKey() []byte {
	return []byte(d.Name) // stub
}
