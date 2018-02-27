package spirit

import (
	"github.com/go-spirit/spirit/component"
	"github.com/go-spirit/spirit/component/function"
	"github.com/go-spirit/spirit/doc"
	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/worker"
	"github.com/gogap/config"
)

type ActorOptions struct {
	url string

	workerDriver string

	componentDriver  string
	componentOptions []component.Option

	config config.Configuration
}

type functionDocument struct {
	document doc.Document
}

func (p *functionDocument) Document() doc.Document {
	return p.document
}

type ActorOption func(*ActorOptions)

func ActorUrl(url string) ActorOption {
	return func(a *ActorOptions) {
		a.url = url
	}
}

func ActorWorker(driver string) ActorOption {
	return func(a *ActorOptions) {
		a.workerDriver = driver
	}
}

func ActorComponent(driver string, opts ...component.Option) ActorOption {
	return func(a *ActorOptions) {
		a.componentDriver = driver
		a.componentOptions = opts
	}
}

func ActorFunction(fn worker.HandlerFunc, opts ...component.Option) ActorOption {
	return func(a *ActorOptions) {
		a.componentDriver = "function"
		a.componentOptions = opts
		a.componentOptions = append(a.componentOptions, function.Function(fn))
	}
}

func ActorFunctionDoc(name string, document doc.Document) ActorOption {
	return func(a *ActorOptions) {
		doc.RegisterDocumenter(name, &functionDocument{document})
	}
}

func ActorConfig(conf config.Configuration) ActorOption {
	return func(a *ActorOptions) {
		a.config = conf
	}
}

type Actor struct {
	url       string
	worker    worker.Worker
	component component.Component
}

func (p *Actor) String() string {
	return p.url
}

func (p *Actor) Url() string {
	return p.url
}

func (p *Actor) Start() error {
	return p.component.Start()
}

func (p *Actor) Stop() error {
	return p.component.Stop()
}

func (p *Actor) Receive(session mail.Session) error {
	h := p.component.Route(session)
	return h(session)
}
