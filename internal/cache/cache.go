package cache

import (
	"errors"
)

// ErrNotFound is returned from typical cache storage mechanisms if a target key isn't stored.
var ErrNotFound = errors.New("key not found in cache")
