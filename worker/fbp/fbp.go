package fbp

import (
	"errors"
	"fmt"
	"sync"

	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/message"
	"github.com/go-spirit/spirit/worker"
	"github.com/go-spirit/spirit/worker/fbp/protocol"
)

var (
	ErrWorkerHasNoPostman = errors.New("invoker has no postman")
)

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

var (
	GraphNameOfError  = "errors"
	GraphNameOfNormal = "normal"
)

func (p *fbpWorker) process(umsg mail.UserMessage) {

	var err error

	session := umsg.Session()
	if session == nil {
		p.EscalateFailure("payload session is nil", umsg)
		return
	}

	payload, ok := session.Payload().Interface().(*protocol.Payload)

	if !ok {
		p.EscalateFailure(
			fmt.Errorf("could not convert session payload to *protocol.Payload"),
			umsg,
		)
		return
	}

	currentGraph := payload.GetCurrentGraph()

	if currentGraph != GraphNameOfError &&
		currentGraph != GraphNameOfNormal {

		p.EscalateFailure(fmt.Errorf("unknown current graph name: '%s'", currentGraph), umsg)
		return
	}

	graph, exist := payload.GetGraph(currentGraph)

	if !exist {
		p.EscalateFailure(
			fmt.Errorf("graph not exist: %s", currentGraph),
			umsg,
		)
		return
	}

	var errH error
	if p.opts.Router != nil {
		handler := p.opts.Router.Route(session)

		if handler != nil {
			errH = handler(session)
			if payload.GetMessage().GetErr() != nil {
				errH = payload.GetMessage().GetErr()
			}
		}
	}

	// nothing todo while session breaked
	if IsSessionBreaked(session) {
		return
	}

	var nextSession mail.Session

	if currentGraph == GraphNameOfNormal && errH != nil {

		errGraph, exist := payload.GetGraph(GraphNameOfError)
		if !exist {
			p.EscalateFailure(
				fmt.Errorf("the normal payload did not have error graph, normal graph name: %s, handler error: %s", currentGraph, errH),
				umsg,
			)
		}

		nextGraph := make(map[string]*protocol.Graph)

		nextGraph[GraphNameOfError] = &protocol.Graph{
			Name:  GraphNameOfError,
			Seq:   0, // it will move forward bellow
			Ports: errGraph.Ports,
		}

		newPayload := &protocol.Payload{
			Id:           payload.GetId(),
			Timestamp:    payload.GetTimestamp(),
			CurrentGraph: GraphNameOfError,
			Graphs:       nextGraph,
			Message: &protocol.Message{
				Id:     payload.GetMessage().GetId(),
				Header: payload.GetMessage().GetHeader(), // TODO need copy
				Body:   payload.GetMessage().GetBody(),
			},
		}

		newPayload.Content().SetError(errH)

		newSession := mail.NewSession()

		newSession.WithPayload(newPayload)

		nextSession = newSession

	} else {
		nextSession = session
	}

	if nextSession == nil {
		return
	}

	current, err := graph.CurrentPort()
	if err != nil {
		p.EscalateFailure(
			fmt.Errorf("get graph current port failure: %s, current graph: %s, seq: %d",
				err,
				currentGraph,
				graph.GetSeq(),
			),
			umsg,
		)
	}

	next, hasNext := graph.NextPort()

	if !hasNext {
		return
	}

	graph.MoveForward()

	nextSession.WithFromTo(current.GetUrl(), next.GetUrl())

	SessionWithPort(nextSession, next.GetUrl(), false, next.GetMetadata())

	err = p.opts.Postman.Post(message.NewUserMessage(nextSession))

	if err != nil {
		p.EscalateFailure(
			fmt.Errorf("post message failure: %s, current graph: %s, seq: %d, to url: %s",
				err,
				currentGraph,
				graph.GetSeq(),
				next.GetUrl(),
			),
			umsg,
		)
		return
	}

	return

}
