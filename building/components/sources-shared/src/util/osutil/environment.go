package osutil

import (
	"os"
	"strings"
)

func ModifiedEnviron(key string, value string) []string {
	env := os.Environ()
	found := false
	for i, entry := range env {
		if strings.HasPrefix(entry, key + "=") {
			env[i] = key + "=" + value
			found = true
		}
	}
	if !found {
		env = append(env, key + "=" + value)
	}
	return env
}
