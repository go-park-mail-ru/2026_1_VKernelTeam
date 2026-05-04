package postgres

import "testing"

func TestNew_BadDSN(t *testing.T) {
	if _, err := New("not-a-valid-dsn"); err == nil {
		t.Fatal("expected error from invalid DSN")
	}
}

func TestNew_DialFails(t *testing.T) {
	if _, err := New("postgres://u:p@127.0.0.1:1/db?sslmode=disable&connect_timeout=1"); err == nil {
		t.Fatal("expected dial error")
	}
}
