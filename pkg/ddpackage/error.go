package ddpackage

import "errors"

var (
  MissingPackJsonError = errors.New("missing pack.json")
	PackJsonReadError = errors.New("pack.json read error")
	InvalidPackJsonError = errors.New("invalid pack.json")
)
