package pgpword

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"strings"
	"testing"
)

func TestSingleRoundTrip(t *testing.T) {
	for _, isEven := range []bool{false, true} {
		for i := 0; i < 256; i++ {
			word := Word(uint8(i), isEven)
			i2, err := Lookup(word, isEven)
			assert.NoError(t, err)
			assert.Equal(t, uint8(i), i2)
		}
	}
}

func TestSample(t *testing.T) {
	// example from the wikipedia page for the PGP wordlist
	input, err := hex.DecodeString("E58294F2E9A227486E8B061B31CC528FD7FA3F19")
	assert.NoError(t, err)
	words := BinToWords(input)
	assert.Equal(t, "topmost Istanbul Pluto vagabond treadmill Pacific brackish dictator goldfish Medusa afflict bravado chatter revolver Dupont midsummer stopwatch whimsical cowbell bottomless", words)
}

// covers (even, odd) and (starts capital, doesn't start capital)
func TestCaseIndependence(t *testing.T) {
	data, err := WordsToBin("FRIGHTEN GRAVITY frighten gravity Frighten Gravity fRIGHTEN gRAVITY FriGhteN GraVitY")
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x68, 0x68, 0x68, 0x68, 0x68, 0x68, 0x68, 0x68, 0x68, 0x68}, data)
	data, err = WordsToBin("MOHAWK JAMAICA mohawk jamaica Mohawk Jamaica mOHAWK jAMAICA MoHawK JamAicA")
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x83, 0x83, 0x83, 0x83, 0x83, 0x83, 0x83, 0x83, 0x83, 0x83}, data)
}

func TestNoWords(t *testing.T) {
	data, err := WordsToBin("")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(data))
}

func TestOneWord(t *testing.T) {
	data, err := WordsToBin("brickyard")
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x2A}, data)
}

func TestOneWordWrong(t *testing.T) {
	_, err := WordsToBin("chambermaid")
	assert.Error(t, err)
}

func TestTwoWords(t *testing.T) {
	data, err := WordsToBin("brickyard chambermaid")
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x2A, 0x2A}, data)
}

func TestBulk(t *testing.T) {
	data := make([]byte, 4096)
	_, _ = rand.New(rand.NewSource(413)).Read(data)
	data2, err := WordsToBin(BinToWords(data))
	assert.NoError(t, err)
	assert.Equal(t, data, data2)
}

func TestSubstring(t *testing.T) {
	data := make([]byte, 123)
	_, _ = rand.New(rand.NewSource(612)).Read(data)
	full := BinToWords(data)
	almostFull := BinToWords(data[:len(data)-1])
	assert.NotEqual(t, full, almostFull)
	assert.True(t, strings.HasPrefix(full, almostFull))
}
