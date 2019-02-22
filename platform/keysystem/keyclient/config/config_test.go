package config

import (
	"reflect"
	"testing"

	"github.com/sipb/homeworld/platform/util/testutil"
)

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig("../testdir/configtest.yaml")
	if err != nil {
		t.Fatal(err)
	}
	expected := Config{
		Keyserver:     "localhost",
		CertPath:      "../testdir/keyclient/granting.pem",
		KeyPath:       "../testdir/keyclient/granting.key",
		AuthorityPath: "../testdir/keyclient/keyservertls.pem",
		TokenPath:     "../testdir/keyclient/bootstrap_token.txt",
		TokenAPI:      "renew-keygrant",
		Downloads: []ConfigDownload{
			{
				Type:    "authority",
				Path:    "../testdir/authorities/etcd-client.pem",
				Name:    "etcd-client",
				Refresh: "24h",
				Mode:    "644",
			},
		},
		Keys: []ConfigKey{
			{
				Name:      "keygranting",
				Type:      "tls",
				Key:       "../testdir/keyclient/granting.key",
				Cert:      "../testdir/keyclient/granting.pem",
				API:       "renew-keygrant",
				InAdvance: "336h",
			},
		},
	}
	if !reflect.DeepEqual(config, expected) {
		t.Error("Mismatch between structs:")
		t.Log(config)
		t.Log(expected)
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	_, err := LoadConfig("../testdir/nonexistent.yaml")
	testutil.CheckError(t, err, "testdir/nonexistent.yaml: no such file or directory")
}

func TestLoadConfig_InvalidData(t *testing.T) {
	_, err := LoadConfig("../testdir/configbroken.yaml")
	testutil.CheckError(t, err, "unmarshal errors:")
	testutil.CheckError(t, err, "field broken not found in struct config.Config")
}
