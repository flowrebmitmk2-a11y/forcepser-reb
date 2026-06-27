package voiceroid2

import "testing"

func TestChooseCharacterName(t *testing.T) {
	got := chooseCharacterName([]characterNameCandidate{
		{name: "標準", count: 2},
		{name: "東北きりたん(v1)", count: 5},
		{name: "ユーザー", count: 2},
	})
	if got != "東北きりたん(v1)" {
		t.Fatalf("unexpected character name: %q", got)
	}
}

func TestShouldIgnoreCharacterName(t *testing.T) {
	cases := map[string]bool{
		"":       true,
		"音声保存":   true,
		"再生時間":   true,
		"19":     true,
		"東北きりたん": false,
	}
	for in, want := range cases {
		if got := shouldIgnoreCharacterName(in); got != want {
			t.Fatalf("unexpected ignore result for %q: got %v want %v", in, got, want)
		}
	}
}
