package component

import (
	"context"
	"fmt"
	"runtime"
	// "path/filepath"
	"reflect"

	"github.com/go-spirit/spirit/cache"
	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/worker"
	"github.com/gogap/config"
)

type Options struct {
	Postman mail.Postman
	Cache   cache.Cache
	Config  config.Configuration

	Context context.Context
}

type Option func(o *Options)

func Postman(man mail.Postman) Option {
	return func(p *Options) {
		p.Postman = man
	}
}

func Cache(c cache.Cache) Option {
	return func(p *Options) {
		p.Cache = c
	}
}

func Config(conf config.Configuration) Option {
	return func(p *Options) {
		p.Config = conf
	}
}

func WithValue(key, val interface{}) Option {
	return func(p *Options) {

		if p.Context == nil {
			p.Context = context.Background()
		}

		p.Context = context.WithValue(p.Context, key, val)
	}
}

type Component interface {
	Start() error
	Stop() error
	Handler() worker.HandlerFunc
}

type NewComponentFunc func(...Option) (Component, error)

var (
	components     map[string]NewComponentFunc = make(map[string]NewComponentFunc)
	componentNames []string
)

type ComponentDescribe struct {
	Name         string
	RegisterFunc string
}

func ListComponents() []ComponentDescribe {

	var desc []ComponentDescribe

	for _, name := range componentNames {
		comp := components[name]

		desc = append(desc, ComponentDescribe{
			Name:         name,
			RegisterFunc: runtime.FuncForPC(reflect.ValueOf(comp).Pointer()).Name(),
		})
	}

	return desc
}

func RegisterComponent(name string, fn NewComponentFunc) {

	if len(name) == 0 {
		panic("component name is empty")
	}

	if fn == nil {
		panic("component fn is nil")
	}

	_, exist := components[name]

	if exist {
		panic(fmt.Sprintf("component already registered: %s", name))
	}

	components[name] = fn
	componentNames = append(componentNames, name)
}

func NewComponent(name string, opts ...Option) (srv Component, err error) {
	fn, exist := components[name]

	if !exist {
		err = fmt.Errorf("component not exist '%s'", name)
		return
	}

	srv, err = fn(opts...)

	return
}
