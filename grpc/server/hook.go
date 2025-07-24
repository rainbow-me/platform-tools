package server

import (
	"context"
	"time"
)

// ShutdownHook represents a function to be executed during graceful shutdown
type ShutdownHook struct {
	Name     string                      // Human-readable name for logging
	Priority int                         // Lower number = higher priority (executed first)
	Timeout  time.Duration               // Maximum time allowed for this hook
	Hook     func(context.Context) error // The actual cleanup function
}

// ShutdownHooks is a sortable slice of shutdown hooks
type ShutdownHooks []ShutdownHook

func (h ShutdownHooks) Len() int           { return len(h) }
func (h ShutdownHooks) Less(i, j int) bool { return h[i].Priority < h[j].Priority }
func (h ShutdownHooks) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
