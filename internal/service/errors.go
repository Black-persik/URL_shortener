package service

import "errors"

var (
	ErrInvalidURL = errors.New("invalid url")
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
)
