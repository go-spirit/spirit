package function

import (
	"context"
	"errors"
	"github.com/go-spirit/spirit/mail"

	"github.com/go-spirit/spirit/component"
	"github.com/go-spirit/spirit/worker"
)

type componentFuncKey struct{}
type componentDocKey struct{}

type componentFunc struct {
	handler worker.HandlerFunc
}

func init() {
	component.RegisterComponent("function", newComponentFunc)
}

func Function(fn worker.HandlerFunc) component.Option {
	return func(o *component.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, componentFuncKey{}, fn)
	}
}

func newComponentFunc(opts ...component.Option) (comp component.Component, err error) {

	compOpts := &component.Options{}

	for _, o := range opts {
		o(compOpts)
	}

	vFn, ok := compOpts.Context.Value(componentFuncKey{}).(worker.HandlerFunc)

	if !ok {
		err = errors.New("function could not convert to worker.HandlerFunc")
		return
	}

	return &componentFunc{
		handler: vFn,
	}, nil
}

func (p *componentFunc) Start() error {
	return nil
}

func (p *componentFunc) Stop() error {
	return nil
}

func (p *componentFunc) Route(mail.Session) worker.HandlerFunc {
	return p.handler
}

// func (p *componentFunc) Document() *doc.Document {
// 	if p.document == nil {
// 		return &doc.Document{
// 			Title:       "Function component - TODO",
// 			Description: "The function component can wrapper anonymous function to this component",
// 		}
// 	}

// 	return p.document
// }
