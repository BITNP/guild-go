package auth

import (
	"testing"
	"time"

	authz "git.bitnp.net/BITNP/bitnp-authz"
)

func TestSessionStoreLifecycle(t *testing.T) {
	store := NewStore(time.Minute)
	permissions := authz.FromRoles([]string{authz.SystemAdminRole})
	id, csrf, err := store.Create("subject", "Name", "mail@example.com", "", permissions)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || csrf == "" || id == csrf {
		t.Fatal("session id and csrf must be distinct non-empty tokens")
	}
	session, ok := store.Get(id)
	if !ok || session.Subject != "subject" || session.Name != "Name" {
		t.Fatalf("session = %+v, ok=%v", session, ok)
	}
	if !session.Permissions.SystemAdmin {
		t.Fatal("permissions were not stored")
	}
	store.Delete(id)
	if _, ok := store.Get(id); ok {
		t.Fatal("deleted session still exists")
	}
}

func TestSessionStoreExpiry(t *testing.T) {
	store := NewStore(20 * time.Millisecond)
	id, _, err := store.Create("subject", "Name", "", "", authz.FromRoles(nil))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(35 * time.Millisecond)
	if _, ok := store.Get(id); ok {
		t.Fatal("expired session still exists")
	}
}
