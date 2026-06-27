package voicepeak

import (
	"path/filepath"
	"testing"
)

func TestBuildExportDir(t *testing.T) {
	wantDir := filepath.Join("testdata", "forcepser")
	dir, err := buildExportDir(func(name, text string) (string, error) {
		return filepath.Join(wantDir, "dummy.wav"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if dir != wantDir {
		t.Fatalf("unexpected export dir: %q", dir)
	}
}
