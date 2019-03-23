package config

type ConfigDownload struct {
	Type    string
	Name    string
	Path    string
	Refresh string
	Mode    string
}

type ConfigKey struct {
	Name      string
	Type      string
	Key       string
	Cert      string
	API       string
	InAdvance string `yaml:"in-advance"`
}

type Config struct {
	AuthorityPath string
	Keyserver     string
	KeyPath       string
	CertPath      string
	TokenPath     string
	TokenAPI      string
	Downloads     []ConfigDownload
	Keys          []ConfigKey
}
