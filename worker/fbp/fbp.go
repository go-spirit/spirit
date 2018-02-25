package fbp

import (
	"errors"
	"fmt"
	"sync"

	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/message"
	"github.com/go-spirit/spirit/protocol"
	"github.com/go-spirit/spirit/worker"
)

var (
	ErrWorkerHasNoPostman = errors.New("invoker has no postman")
)

// type CtxKeyPayload struct{}

type ctxKeyPort struct{}
type ctxValuePort struct {
	IsErrorPort bool
	Url         string
	Metadata    map[string]string
}

type ctxKeyBreakSession struct{}

type fbpWorker struct {
	opts worker.WorkerOptions

	lock  sync.Mutex
	stopC chan bool
}

func init() {
	worker.RegisterWorker("fbp", NewFBPWorker)
}

func BreakSession(s mail.Session) {

	if IsSessionBreaked(s) {
		return
	}

	s.WithValue(ctxKeyBreakSession{}, true)
}

func IsSessionBreaked(s mail.Session) bool {
	breaked, ok := s.Value(ctxKeyBreakSession{}).(bool)
	if ok {
		return breaked
	}

	return false
}

func SessionWithPort(s mail.Session, url string, isErrorPort bool, metadata map[string]string) {
	s.WithValue(
		ctxKeyPort{},
		&ctxValuePort{
			isErrorPort, url, metadata,
		},
	)
}

func GetSessionPort(s mail.Session) *ctxValuePort {
	port, ok := s.Value(ctxKeyPort{}).(*ctxValuePort)
	if !ok {
		return nil
	}

	return port
}

func NewFBPWorker(opts ...worker.WorkerOption) (worker worker.Worker, err error) {

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

func (p *fbpWorker) Init(opts ...worker.WorkerOption) {
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

	sessionPort := GetSessionPort(session)
	if sessionPort == nil {
		err = errors.New("session did not have port info")
		return
	}

	// errors info already processed, do not post to next
	if sessionPort.IsErrorPort {
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

		if IsSessionBreaked(session) {
			fmt.Println("breaked")
			return
		}
		fmt.Println("not breaked")

		next, hasNext := graph.NextPort()

		if !hasNext {
			return
		}

		err = graph.MoveForward()
		if err != nil {
			return
		}

		session.WithFromTo(current.GetUrl(), next.GetUrl())

		SessionWithPort(session, next.GetUrl(), false, next.GetMetadata())

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

			newSession := session.Fork()
			newSession.WithFromTo(currentURL, to)

			SessionWithPort(newSession, to, true, nextPorts[i].GetMetadata())

			umsg := message.NewUserMessage(newSession)
			err = p.opts.Postman.Post(umsg)
			return
		}
	}

	return
}
