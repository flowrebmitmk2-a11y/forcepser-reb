package voicepeak

import (
	"path/filepath"
	"testing"
)

func TestBuildExportDir(t *testing.T) {
	dir, err := buildExportDir(func(name, text string) (string, error) {
		return filepath.Join(`C:\tmp\forcepser`, "dummy.wav"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if dir != `C:\tmp\forcepser` {
		t.Fatalf("unexpected export dir: %q", dir)
	}
}
