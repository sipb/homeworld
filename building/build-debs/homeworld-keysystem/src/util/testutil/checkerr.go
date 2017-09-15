package testutil

import (
	"runtime"
	"strings"
	"testing"
)

// Not really unit-testable...
func CheckError(t *testing.T, err error, substr string) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		segs := strings.Split(file, "/")
		file = segs[len(segs)-1]
	} else {
		file, line = "???", 1
	}
	if err == nil {
		t.Fatalf("at %s:%d: expected error; found no error", file, line)
	} else if !strings.Contains(err.Error(), substr) {
		t.Fatalf("at %s:%d: expected error that contains '%s', but only got '%s'", file, line, substr, err.Error())
	}
}
