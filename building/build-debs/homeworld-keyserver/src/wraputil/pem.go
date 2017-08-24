package wraputil

import (
	"bytes"
	"encoding/pem"
	"errors"
	"fmt"
)

func LoadSinglePEMBlock(data []byte, expected_types []string) ([]byte, error) {
	if !bytes.HasPrefix(data, []byte("-----BEGIN ")) {
		return nil, errors.New("Missing expected PEM header")
	}
	pemBlock, remain := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("Could not parse PEM data")
	}
	found := false
	for _, expected_type := range expected_types {
		if pemBlock.Type == expected_type {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("Found PEM block of type \"%s\" instead of types %s", pemBlock.Type, expected_types)
	}
	if remain != nil && len(remain) > 0 {
		return nil, errors.New("Trailing data found after PEM data")
	}
	return pemBlock.Bytes, nil
}
