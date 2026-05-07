package biz

import "errors"

var (
	ErrForbidden    = errors.New("forbidden")
	ErrBadParam     = errors.New("bad param")
	ErrNoPermission = errors.New("no permission")
)
