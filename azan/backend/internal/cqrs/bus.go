// Package cqrs provides minimal in-process command and query buses.
// Commands mutate state (write side), queries read projections (read side).
package cqrs

import (
	"context"
	"fmt"
	"sync"
)

// Command is a request to change state. Handlers may return a result payload
// (e.g. the id of a created aggregate) alongside the emitted events.
type Command interface {
	CommandName() string
}

// Query is a request to read state. It must never mutate anything.
type Query interface {
	QueryName() string
}

type CommandHandler func(ctx context.Context, cmd Command) (any, error)
type QueryHandler func(ctx context.Context, q Query) (any, error)

type CommandBus struct {
	mu       sync.RWMutex
	handlers map[string]CommandHandler
}

func NewCommandBus() *CommandBus {
	return &CommandBus{handlers: map[string]CommandHandler{}}
}

func (b *CommandBus) Register(name string, h CommandHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = h
}

func (b *CommandBus) Dispatch(ctx context.Context, cmd Command) (any, error) {
	b.mu.RLock()
	h, ok := b.handlers[cmd.CommandName()]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("cqrs: no handler for command %q", cmd.CommandName())
	}
	return h(ctx, cmd)
}

type QueryBus struct {
	mu       sync.RWMutex
	handlers map[string]QueryHandler
}

func NewQueryBus() *QueryBus {
	return &QueryBus{handlers: map[string]QueryHandler{}}
}

func (b *QueryBus) Register(name string, h QueryHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = h
}

func (b *QueryBus) Dispatch(ctx context.Context, q Query) (any, error) {
	b.mu.RLock()
	h, ok := b.handlers[q.QueryName()]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("cqrs: no handler for query %q", q.QueryName())
	}
	return h(ctx, q)
}
