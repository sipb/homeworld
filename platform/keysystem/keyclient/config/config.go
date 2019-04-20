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
