package authorities

type Authority interface {
	GetPublicKey() []byte
}
