package nodes

import (
	"strings"
	"testing"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
)

func TestNormalizeNodeFields(t *testing.T) {
	title, description, tags, err := normalizeNodeFields("  title  ", "  desc  ", []string{" a ", "a", "", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if title != "title" || description != "desc" || strings.Join(tags, ",") != "a,b" {
		t.Fatalf("normalized = %q %q %v", title, description, tags)
	}
	if _, _, _, err := normalizeNodeFields("", "", nil); apperr.From(err).Status != 400 {
		t.Fatalf("empty title status = %d", apperr.From(err).Status)
	}
	if _, _, _, err := normalizeNodeFields("title", "", []string{strings.Repeat("x", 33)}); apperr.From(err).Status != 400 {
		t.Fatalf("long tag status = %d", apperr.From(err).Status)
	}
	tooMany := make([]string, 11)
	for index := range tooMany {
		tooMany[index] = string(rune('a' + index))
	}
	if _, _, _, err := normalizeNodeFields("title", "", tooMany); apperr.From(err).Status != 400 {
		t.Fatalf("too many tags status = %d", apperr.From(err).Status)
	}
}

func TestRelationFor(t *testing.T) {
	leaders := map[int64]struct{}{1: {}}
	members := map[int64]struct{}{1: {}, 2: {}}
	if value := relationFor(1, leaders, members); value == nil || *value != "leader" {
		t.Fatalf("leader relation = %v", value)
	}
	if value := relationFor(2, leaders, members); value == nil || *value != "member" {
		t.Fatalf("member relation = %v", value)
	}
	if value := relationFor(3, leaders, members); value != nil {
		t.Fatalf("unrelated relation = %v", value)
	}
}
