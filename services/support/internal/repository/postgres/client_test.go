package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_InvalidDSN(t *testing.T) {
	_, err := New("not a valid dsn://")
	assert.Error(t, err)
}

func TestNew_UnreachableDB(t *testing.T) {
	_, err := New("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
	assert.Error(t, err)
}
