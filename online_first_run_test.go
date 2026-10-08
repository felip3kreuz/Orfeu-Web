package main

import "testing"

func TestNeedsFirstRunSetup(t *testing.T) {
	st := newServerState(t.TempDir())
	if !needsFirstRunSetup(st) {
		t.Fatal("servidor vazio deveria exigir configuração inicial")
	}

	if _, err := st.createInitialAdmin("Admin", "admin@example.com", "SenhaAdmin123"); err != nil {
		t.Fatal(err)
	}
	if needsFirstRunSetup(st) {
		t.Fatal("servidor com administrador não deveria exigir configuração inicial")
	}
}
