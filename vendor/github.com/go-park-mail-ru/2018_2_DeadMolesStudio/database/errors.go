package database

import (
	"errors"
)

var (
	ErrNotNullConstraintViolation = errors.New("not null constraint violation")
	ErrUniqueConstraintViolation  = errors.New("unique constraint violation")
)
