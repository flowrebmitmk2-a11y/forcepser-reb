package voiceroid2

import "testing"

func TestNormalizeMenuCaption(t *testing.T) {
	cases := map[string]string{
		"ファイル(&F)":     "ファイル",
		"ファイル(_F)":     "ファイル",
		"音声保存(&S)...":  "音声保存",
		"音声保存\tCtrl+S": "音声保存",
		"音声ファイルを保存...": "音声ファイルを保存",
		"音声ファイルを保存 後ろ": "音声ファイルを保存 後ろ",
	}
	for in, want := range cases {
		if got := normalizeMenuCaption(in); got != want {
			t.Fatalf("unexpected caption normalize: %q -> %q want %q", in, got, want)
		}
	}
}

func TestMenuCaptionMatches(t *testing.T) {
	if menuCaptionMatches("テキストを保存", saveMenuItemCaptions) {
		t.Fatal("text save should not match audio save captions")
	}
	if !menuCaptionHasPrefix("音声ファイルを保存...", saveMenuItemCaptions) {
		t.Fatal("audio save caption should match")
	}
	if !menuCaptionHasPrefix("音声ファイルを保存 連番付き", saveMenuItemCaptions) {
		t.Fatal("audio save caption with trailing text should match")
	}
	if menuCaptionHasPrefix("テキストを保存", saveMenuItemCaptions) {
		t.Fatal("text save should not match audio save caption prefix")
	}
	if !menuCaptionHasPrefix("ファイル(&F)", fileMenuItemCaptions) {
		t.Fatal("file menu caption with access key should match by prefix")
	}
	if !menuCaptionHasPrefix("ファイル(_F)", fileMenuItemCaptions) {
		t.Fatal("file menu caption with underscore access key should match by prefix")
	}
}
