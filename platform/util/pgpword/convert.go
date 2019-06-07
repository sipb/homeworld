package pgpword

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

func Word(index uint8, isEven bool) string {
	if isEven {
		return evenWords[index]
	} else {
		return oddWords[index]
	}
}

type lookup struct {
	index  uint8
	isEven bool
}

var lookups map[string]lookup

func init() {
	lookups = make(map[string]lookup)
	for i, word := range evenWords {
		if int(uint8(i)) != i {
			panic("unexpected overflow on uint8")
		}
		lookups[strings.ToLower(word)] = lookup{uint8(i), true}
	}
	for i, word := range oddWords {
		if int(uint8(i)) != i {
			panic("unexpected overflow on uint8")
		}
		lookups[strings.ToLower(word)] = lookup{uint8(i), false}
	}
}

func LookupEither(word string) (index uint8, isEven bool, err error) {
	lookup, ok := lookups[strings.ToLower(word)]
	if !ok {
		return 0, false, fmt.Errorf("invalid PGP word: '%s'", word)
	} else {
		return lookup.index, lookup.isEven, nil
	}
}

func Lookup(word string, isEven bool) (uint8, error) {
	index, foundEven, err := LookupEither(word)
	if err != nil {
		return 0, err
	}
	if foundEven != isEven {
		return 0, errors.New("PGP word parity mismatch: word missing or misordered")
	}
	return index, nil
}

func BinToWords(data []byte) string {
	words := make([]string, len(data))
	for i, b := range data {
		words[i] = Word(b, i%2 == 0)
	}
	return strings.Join(words, " ")
}

func WordsToBin(text string) ([]byte, error) {
	words := strings.Fields(text)
	bytes := make([]byte, len(words))
	for i, word := range words {
		b, err := Lookup(word, i%2 == 0)
		if err != nil {
			return nil, err
		}
		bytes[i] = b
	}
	return bytes, nil
}
