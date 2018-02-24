package context

import (
	"context"
	"sync"
	"time"
)

type ctxKeyError struct{}

type Context interface {
	Value(key interface{}) interface{}
	WithValue(key, val interface{})

	Err() error
	WithError(error)

	WithCancel() (cancel context.CancelFunc)
	WithDeadline(deadline time.Time) (cancel context.CancelFunc)
	WithTimeout(timeout time.Duration) (cancel context.CancelFunc)
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}

	Copy() Context
}

type defaultContext struct {
	ctx context.Context

	keys []interface{}
	loc  sync.RWMutex
}

func NewContext() Context {
	return &defaultContext{
		ctx: context.Background(),
	}
}

func (p *defaultContext) WithValue(key, val interface{}) {
	p.loc.Lock()
	p.ctx = context.WithValue(p.ctx, key, val)
	p.keys = append(p.keys, key)
	p.loc.Unlock()
}

func (p *defaultContext) Value(key interface{}) interface{} {
	p.loc.RLock()
	ret := p.ctx.Value(key)
	p.loc.RUnlock()

	return ret
}

func (p *defaultContext) Err() error {
	p.loc.RLock()
	defer p.loc.RUnlock()

	v, ok := p.ctx.Value(ctxKeyError{}).(error)
	if ok {
		return v
	}

	return p.ctx.Err()
}

func (p *defaultContext) WithCancel() (cancel context.CancelFunc) {
	p.loc.Lock()
	p.ctx, cancel = context.WithCancel(p.ctx)
	p.loc.Unlock()

	return
}

func (p *defaultContext) WithDeadline(deadline time.Time) (cancel context.CancelFunc) {
	p.loc.Lock()
	p.ctx, cancel = context.WithDeadline(p.ctx, deadline)
	p.loc.Unlock()

	return
}

func (p *defaultContext) WithTimeout(timeout time.Duration) (cancel context.CancelFunc) {
	p.loc.Lock()
	p.ctx, cancel = context.WithTimeout(p.ctx, timeout)
	p.loc.Unlock()

	return
}

func (p *defaultContext) WithError(err error) {
	p.WithValue(ctxKeyError{}, err)
}

func (p *defaultContext) Deadline() (deadline time.Time, ok bool) {
	return p.ctx.Deadline()
}

func (p *defaultContext) Done() <-chan struct{} {
	return p.ctx.Done()
}

func (p *defaultContext) Copy() Context {
	p.loc.RLock()
	defer p.loc.RUnlock()

	ctx := context.Background()

	for i := 0; i < len(p.keys); i++ {

		v := p.Value(p.keys[i])

		ctx = context.WithValue(ctx, p.keys[i], v)
	}

	if p.Err() != nil {
		ctx = context.WithValue(ctx, ctxKeyError{}, p.Err())
	}

	return &defaultContext{
		ctx: ctx,
	}
}
