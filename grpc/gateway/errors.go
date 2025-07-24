package gateway

import "github.com/pkg/errors"

var (
	ErrNoEndpointsRegistered = errors.Errorf("no Endpoints registered")
	ErrInvalidPrefix         = errors.New("invalid prefix")
)
