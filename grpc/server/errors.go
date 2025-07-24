package server

import "errors"

var ErrShutdownTimeout = errors.New("shutdown timed out")
