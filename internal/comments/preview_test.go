package comments

import "testing"

func TestTextPreview(t *testing.T) {
	if got := TextPreview("  hello  ", 10); got != "hello" {
		t.Fatalf("trim = %q", got)
	}
	if got := TextPreview("一二三四五六", 4); got != "一二三四…" {
		t.Fatalf("truncate = %q", got)
	}
	if got := TextPreview("short", 10); got != "short" {
		t.Fatalf("short = %q", got)
	}
}

func TestValidEmoji(t *testing.T) {
	for _, key := range EmojiKeys {
		if !ValidEmoji(key) {
			t.Fatalf("%s should be valid", key)
		}
	}
	if ValidEmoji("unknown") {
		t.Fatal("unknown emoji should be rejected")
	}
}
