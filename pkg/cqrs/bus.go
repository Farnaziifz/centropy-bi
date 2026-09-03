// Package cqrs is a minimal, dependency-free command/query bus.
//
// It uses Go generics instead of reflection-based method dispatch so that
// registering and executing a handler is a single map lookup keyed by the
// message's concrete type — no runtime type assertions on the hot path
// beyond that lookup, and full compile-time type safety for callers.
package cqrs

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// CommandHandlerFunc handles a command of type C and returns a result of type R.
type CommandHandlerFunc[C any, R any] func(ctx context.Context, cmd C) (R, error)

// QueryHandlerFunc handles a query of type Q and returns a result of type R.
type QueryHandlerFunc[Q any, R any] func(ctx context.Context, q Q) (R, error)

// erasedHandler is the type-erased form every handler is stored as, so
// dispatch never has to deal with named generic func types at runtime.
type erasedHandler func(ctx context.Context, msg any) (any, error)

// Bus is a process-local, in-memory command/query dispatcher. One Bus
// instance is safe for concurrent use and is shared for the process
// lifetime; registration happens once at startup (see cmd/api/main.go).
type Bus struct {
	mu       sync.RWMutex
	handlers map[reflect.Type]erasedHandler
}

// NewBus builds an empty Bus.
func NewBus() *Bus {
	return &Bus{handlers: make(map[reflect.Type]erasedHandler)}
}

// RegisterCommand wires a handler for command type C. Panics on duplicate
// registration — that is always a wiring bug caught at startup, never at
// request time.
func RegisterCommand[C any, R any](bus *Bus, handler CommandHandlerFunc[C, R]) {
	register[C, R](bus, func(ctx context.Context, cmd C) (R, error) {
		return handler(ctx, cmd)
	})
}

// RegisterQuery wires a handler for query type Q.
func RegisterQuery[Q any, R any](bus *Bus, handler QueryHandlerFunc[Q, R]) {
	register[Q, R](bus, func(ctx context.Context, q Q) (R, error) {
		return handler(ctx, q)
	})
}

func register[M any, R any](bus *Bus, handler func(ctx context.Context, msg M) (R, error)) {
	key := reflect.TypeOf((*M)(nil)).Elem()

	erased := func(ctx context.Context, msg any) (any, error) {
		return handler(ctx, msg.(M))
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()
	if _, exists := bus.handlers[key]; exists {
		panic(fmt.Sprintf("cqrs: duplicate handler registration for %s", key))
	}
	bus.handlers[key] = erased
}

// ExecuteCommand dispatches cmd to its registered handler.
func ExecuteCommand[C any, R any](ctx context.Context, bus *Bus, cmd C) (R, error) {
	return dispatch[C, R](ctx, bus, cmd)
}

// ExecuteQuery dispatches q to its registered handler.
func ExecuteQuery[Q any, R any](ctx context.Context, bus *Bus, q Q) (R, error) {
	return dispatch[Q, R](ctx, bus, q)
}

func dispatch[M any, R any](ctx context.Context, bus *Bus, msg M) (R, error) {
	var zero R
	key := reflect.TypeOf((*M)(nil)).Elem()

	bus.mu.RLock()
	handler, ok := bus.handlers[key]
	bus.mu.RUnlock()
	if !ok {
		return zero, fmt.Errorf("cqrs: no handler registered for %s", key)
	}

	result, err := handler(ctx, msg)
	if err != nil {
		return zero, err
	}
	r, ok := result.(R)
	if !ok {
		return zero, fmt.Errorf("cqrs: handler for %s returned an incompatible result type", key)
	}
	return r, nil
}
