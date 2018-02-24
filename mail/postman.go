package mail

import (
	"fmt"
)

type PostmanOptions struct {
	Registry Registry

	inboundMiddleware InboundMiddlewareFunc
}

type PostmanOption func(*PostmanOptions)

func PostmanRegistry(reg Registry) PostmanOption {
	return func(p *PostmanOptions) {
		p.Registry = reg
	}
}

func InboundMiddleware(fn InboundMiddlewareFunc) PostmanOption {
	return func(p *PostmanOptions) {
		p.inboundMiddleware = fn
	}
}

type Inbound interface {
	PostSystemMessage(interface{})
	PostUserMessage(interface{})
}

type Postman interface {
	Post(Message) error

	Registry() Registry
	Start() error
}

type NewPostmanFunc func(...PostmanOption) (Postman, error)

var (
	postmen map[string]NewPostmanFunc = make(map[string]NewPostmanFunc)
)

func RegisterPostman(name string, fn NewPostmanFunc) {
	if len(name) == 0 {
		panic("postman name is empty")
	}

	if fn == nil {
		panic("postman fn is nil")
	}

	_, exist := postmen[name]

	if exist {
		panic(fmt.Sprintf("postman already registered: %s", name))
	}

	postmen[name] = fn
}

func NewPostman(name string, opts ...PostmanOption) (man Postman, err error) {
	fn, exist := postmen[name]

	if !exist {
		err = fmt.Errorf("postman not exist '%s'", name)
		return
	}

	man, err = fn(opts...)

	return
}
