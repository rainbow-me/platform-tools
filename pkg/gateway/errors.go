package gateway

import "github.com/pkg/errors"

var (
	ErrNoEndpointsRegistered = errors.Errorf("no endpoints registered")
	ErrInvalidPrefix         = errors.New("invalid prefix")
)
