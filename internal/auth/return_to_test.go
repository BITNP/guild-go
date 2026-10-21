package auth

import "testing"

func TestSafeReturnTo(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"root", "/", "/"},
		{"path", "/nodes/12?tab=comments", "/nodes/12?tab=comments"},
		{"empty", "", "/"},
		{"external", "https://example.com/", "/"},
		{"protocol relative", "//example.com/", "/"},
		{"backslash", "/\\example.com", "/"},
		{"api", "/api/nodes", "/"},
		{"login", "/login", "/"},
		{"control", "/nodes/1\n", "/"},
		{"encoded api", "/%61pi/nodes", "/"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := SafeReturnTo(test.in); got != test.want {
				t.Fatalf("SafeReturnTo(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestSafePictureURL(t *testing.T) {
	if got := SafePictureURL("https://example.com/a.png"); got != "https://example.com/a.png" {
		t.Fatalf("valid picture rejected: %q", got)
	}
	for _, value := range []string{"", "javascript:alert(1)", "data:image/png;base64,xxx", "/relative.png"} {
		if got := SafePictureURL(value); got != "" {
			t.Fatalf("unsafe picture accepted: %q -> %q", value, got)
		}
	}
}
