package database

import (
	"errors"
)

var (
	ErrConnRefused = errors.New("conn is not opened")

	ErrNotNullConstraintViolation = errors.New("not null constraint violation")
	ErrUniqueConstraintViolation  = errors.New("unique constraint violation")
)
