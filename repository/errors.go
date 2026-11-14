package repository

import "errors"

var (
	ErrNotFound       = errors.New("repo error not found")
	ErrNoAffectedRows = errors.New("repo error no affected rows")
	ErrConflict       = errors.New("repo error conflict")
)
