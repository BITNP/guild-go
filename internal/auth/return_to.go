package auth

import (
	"net/url"
	"strings"
)

// SafeReturnTo restricts post-login redirects to same-site application paths.
func SafeReturnTo(value string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/"
	}
	if strings.ContainsAny(value, "\\\r\n\t") {
		return "/"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/"
	}
	path := parsed.Path
	if path == "/login" || strings.HasPrefix(path, "/api/") {
		return "/"
	}
	return value
}
