package session

import (
	"errors"
)

var (
	ErrKeyNotFound = errors.New("key not found")

	ErrConnRefused = errors.New("conn is not opened")
)
