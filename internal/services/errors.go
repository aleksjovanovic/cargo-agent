package services

import "errors"

var (
	ErrNotFound     = errors.New("not_found")
	ErrConflict     = errors.New("conflict")
	ErrInvalid      = errors.New("invalid")
	ErrUnauthorized = errors.New("unauthorized")
	ErrNoChanges    = errors.New("no_changes")
)
