package config

type ConfigAuthority struct {
	Type      string
	Key       string
	Cert      string
	PresentAs []string
}
