package worker

import (
	"errors"
	"sync"

	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/message"
	"github.com/go-spirit/spirit/protocol"
)

var (
	ErrWorkerHasNoPostman = errors.New("invoker has no postman")
)

type CtxKeyPayload struct{}

type CtxKeyPort struct{}
type CtxValuePort struct {
	To       string
	Metadata map[string]string
}

type fbpWorker struct {
	opts WorkerOptions

	lock  sync.Mutex
	stopC chan bool
}

func init() {
	RegisterWorker("fbp", NewFBPWorker)
}

func NewFBPWorker(opts ...WorkerOption) (worker Worker, err error) {

	a := &fbpWorker{
		stopC: make(chan bool),
	}

	for _, o := range opts {
		o(&a.opts)
	}

	if a.opts.Postman == nil {
		err = ErrWorkerHasNoPostman
		return
	}

	worker = a

	return
}

func (p *fbpWorker) Init(opts ...WorkerOption) {
	for _, o := range opts {
		o(&p.opts)
	}
}

func (p *fbpWorker) process(umsg mail.UserMessage) {

	session := umsg.Session()

	hasErrorBefore := session.Err() != nil

	payload := session.Payload().(*protocol.Payload)

	graph := payload.GetGraph()

	if graph == nil {
		p.EscalateFailure("Payload graph is empty", umsg)
		return
	}

	var errH error
	if p.opts.Handler != nil {
		errH = p.opts.Handler(session)
		if payload.GetMessage().GetError() != nil {
			errH = payload.GetMessage().GetError()
		}
	}

	if !hasErrorBefore {
		err := p.moveForwardAndPost(umsg, errH)
		if err != nil {
			p.EscalateFailure(err, umsg)
			return
		}
	}

	return
}

func (p *fbpWorker) moveForwardAndPost(m mail.UserMessage, e error) (err error) {

	session := m.Session()

	if session == nil {
		err = errors.New("session is nil")
		return
	}

	if session.Payload() == nil {
		err = errors.New("payload is nil")
		return
	}

	payload, ok := session.Payload().(*protocol.Payload)

	if !ok {
		err = errors.New("could not convert session's content to payload")
		return
	}

	if e != nil {
		session.WithError(e)
		payload.GetMessage().SetError(e)
	}

	graph := payload.GetGraph()

	current, err := graph.CurrentPort()
	if err != nil {
		return
	}

	if session.Err() == nil {

		next, hasNext := graph.NextPort()

		if !hasNext {
			return
		}

		err = graph.MoveForward()
		if err != nil {
			return
		}

		session.WithFromTo(current.GetUrl(), next.GetUrl())

		session.WithValue(
			CtxKeyPort{},
			&CtxValuePort{
				To:       next.GetUrl(),
				Metadata: next.GetMetadata(),
			})

		err = p.opts.Postman.Post(message.NewUserMessage(session))
		return

	} else {

		nextPorts := graph.GetErrors()
		currentURL := current.GetUrl()

		for i := 0; i < len(nextPorts); i++ {

			to := nextPorts[i].GetUrl()

			if to == currentURL {
				continue
			}

			newEnv := session.Fork()
			newEnv.WithFromTo(currentURL, to)
			newEnv.WithValue(
				CtxKeyPort{},
				&CtxValuePort{
					To:       to,
					Metadata: nextPorts[i].GetMetadata(),
				},
			)

			umsg := message.NewUserMessage(newEnv)
			err = p.opts.Postman.Post(umsg)
			return
		}
	}

	return
}
