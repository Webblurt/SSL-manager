package models

import "errors"

var (
	ErrDomainExists   = errors.New("domain already exists")
	ErrDomainNotFound = errors.New("domain not found")
)
