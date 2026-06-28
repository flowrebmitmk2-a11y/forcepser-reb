package voiceroid2

import (
	"strconv"
	"strings"
	"testing"
)

func TestBuildSaveFileName(t *testing.T) {
	SetSaveFileNamePrefix("")
	got := buildSaveFileName()
	if !strings.HasPrefix(got, saveFileNamePrefix+"_") {
		t.Fatalf("unexpected prefix: %q", got)
	}
	if !strings.HasSuffix(got, ".wav") {
		t.Fatalf("unexpected suffix: %q", got)
	}
	millis := strings.TrimSuffix(strings.TrimPrefix(got, saveFileNamePrefix+"_"), ".wav")
	if _, err := strconv.ParseInt(millis, 10, 64); err != nil {
		t.Fatalf("timestamp is not numeric: %q", got)
	}
}

func TestSetSaveFileNamePrefix(t *testing.T) {
	SetSaveFileNamePrefix("custom")
	if saveFileNamePrefix != "custom" {
		t.Fatalf("unexpected custom prefix: %q", saveFileNamePrefix)
	}
	SetSaveFileNamePrefix("")
	if saveFileNamePrefix != defaultSaveFileNamePrefix {
		t.Fatalf("unexpected default prefix: %q", saveFileNamePrefix)
	}
}
