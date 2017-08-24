package wraputil

import (
	"testing"
	"testutil"
)

func TestLoadSinglePEMBlock_CompletelyWrong(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("this isn't a pem block"), []string{"TEST"})
	testutil.CheckError(t, err, "Missing expected PEM header")
}

func TestLoadSinglePEMBlock_Malformed(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN PEM BLOCK-----\nTHIS PEM BLOCK IS MALFORMED\n-----END PEM BLOCK-----\n"), []string{"TEST"})
	testutil.CheckError(t, err, "Could not parse PEM data")
}

func TestLoadSinglePEMBlock_MismatchEnds(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END PEM BLOCK-----\n"), []string{"TEST"})
	testutil.CheckError(t, err, "Could not parse PEM data")
}

func TestLoadSinglePEMBlock_TrailingData(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END TEST-----\nTrailing data\n"), []string{"TEST"})
	testutil.CheckError(t, err, "Trailing data")
}

func TestLoadSinglePEMBlock_TrailingNewlines(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END TEST-----\n\n"), []string{"TEST"})
	testutil.CheckError(t, err, "Trailing data")
}

func TestLoadSinglePEMBlock_EmptyTypes(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END TEST-----\n"), nil)
	testutil.CheckError(t, err, "instead of types []")
}

func TestLoadSinglePEMBlock_EmptyTypes2(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN -----\nAAAA\n-----END -----\n"), nil)
	testutil.CheckError(t, err, "instead of types []")
}

func TestLoadSinglePEMBlock_FirstType(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST1-----\nAAAA\n-----END TEST1-----\n"), []string{"TEST1", "TEST2"})
	if err != nil {
		t.Error(err)
	}
}

func TestLoadSinglePEMBlock_SecondType(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST2-----\nAAAA\n-----END TEST2-----\n"), []string{"TEST1", "TEST2"})
	if err != nil {
		t.Error(err)
	}
}

func TestLoadSinglePEMBlock_NeitherType(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST3-----\nAAAA\n-----END TEST3-----\n"), []string{"TEST1", "TEST2"})
	testutil.CheckError(t, err, "of type \"TEST3\" instead of types [TEST1 TEST2]")
}

func TestLoadSinglePEMBlock_CorrectResult(t *testing.T) { // not really a supported configuration, but make sure it fails
	out, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END TEST-----\n"), []string{"TEST"})
	if err != nil {
		t.Error(err)
	} else {
		if string(out) != "\x00\x00\x00" {
			t.Error("Wrong result.")
		}
	}
}
