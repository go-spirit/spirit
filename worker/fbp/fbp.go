package fbp

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/message"
	"github.com/go-spirit/spirit/worker"
	"github.com/go-spirit/spirit/worker/fbp/protocol"
	"github.com/sirupsen/logrus"
)

var (
	ErrWorkerHasNoPostman  = errors.New("invoker has no postman")
	ErrExecuteHandlerPanic = errors.New("execute handler panic")
)

var (
	GraphNameOfError      = "error"
	GraphNameOfEntrypoint = "entrypoint"
)

type ctxKeyPort struct{}
type ctxValuePort struct {
	CurrentGraph string
	Url          string
	Metadata     map[string]string
}

type ctxKeyBreakSession struct{}
type ctxKeySwitchGraph struct{}

type fbpWorker struct {
	opts worker.WorkerOptions

	lock  sync.Mutex
	stopC chan bool
}

func init() {
	worker.RegisterWorker("fbp", NewFBPWorker)
}

func SwitchGraph(session mail.Session, toGraph string) (err error) {

	if session == nil {
		err = errors.New("payload session is nil")
		return
	}

	if len(toGraph) == 0 {
		err = errors.New("graph name is emtpy")
		return
	}

	if toGraph == GraphNameOfError {
		err = errors.New("could not switch graph to error by manual, just return error in component handler")
		return
	}

	payload, ok := session.Payload().Interface().(*protocol.Payload)

	if !ok {
		err = fmt.Errorf("could not convert session payload to *protocol.Payload")
		return
	}

	currentGraph := payload.GetCurrentGraph()

	if currentGraph == toGraph {
		return
	}

	graphs := payload.GetGraphs()

	_, exist := graphs[toGraph]
	if !exist {
		err = fmt.Errorf("graph %s not exist, switch graph failure", toGraph)
		return
	}

	session.WithValue(ctxKeySwitchGraph{}, toGraph)

	return
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

func SessionWithPort(s mail.Session, graph, url string, metadata map[string]string) {
	s.WithValue(
		ctxKeyPort{},
		&ctxValuePort{
			graph, url, metadata,
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

func GetSwitchGraphName(s mail.Session) string {
	name, ok := s.Value(ctxKeySwitchGraph{}).(string)
	if !ok {
		return ""
	}

	return name
}

func PartitionFromSession(s mail.Session, entrypointGraphName, errorGraphName string, otherGraphNames ...string) (session mail.Session, err error) {

	if len(entrypointGraphName) == 0 || len(errorGraphName) == 0 {
		err = errors.New("graph name could not be empty")
		return
	}

	originPayload, ok := s.Payload().Interface().(*protocol.Payload)
	if !ok {
		err = errors.New("could not convert session payload to *protocol.Payload")
		return
	}

	originCurrentGraph, exist := originPayload.GetGraph(originPayload.GetCurrentGraph())
	if !exist {
		err = fmt.Errorf("graph of %s not in orignal payload", originPayload.GetCurrentGraph())
		return
	}

	entrypointGraph, exist := originPayload.GetGraph(entrypointGraphName)
	if !exist {
		err = fmt.Errorf("graph of %s not exist", entrypointGraphName)
		return
	}

	errorGraph, exist := originPayload.GetGraph(errorGraphName)
	if !exist {
		err = fmt.Errorf("graph of %s not exist", errorGraphName)
		return
	}

	newGraphs := map[string]*protocol.Graph{
		GraphNameOfEntrypoint: entrypointGraph.CopyAndRename(GraphNameOfEntrypoint),
		GraphNameOfError:      errorGraph.CopyAndRename(GraphNameOfError),
	}

	for _, graphName := range otherGraphNames {
		g, exist := originPayload.GetGraph(errorGraphName)
		if !exist {
			err = fmt.Errorf("graph of %s not exist", graphName)
			return
		}

		newGraphs[graphName] = g.Copy()
	}

	payload := &protocol.Payload{
		Id:           originPayload.GetId(),
		Timestamp:    time.Now().UnixNano(),
		CurrentGraph: GraphNameOfEntrypoint,
		Graphs:       newGraphs,
		Message:      originPayload.Message.Copy().(*protocol.Message),
	}

	port, err := entrypointGraph.CurrentPort()
	if err != nil {
		return
	}

	originCurrentPort, err := originCurrentGraph.CurrentPort()
	if err != nil {
		return
	}

	session = mail.NewSession()
	session.WithPayload(payload)
	session.WithFromTo(originCurrentPort.GetUrl(), port.GetUrl())

	SessionWithPort(session, entrypointGraph.GetName(), port.GetUrl(), port.GetMetadata())

	return
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

type fbpMessage struct {
	Session         mail.Session
	Payload         *protocol.Payload
	CurrentGraph    *protocol.Graph
	NextGraph       *protocol.Graph
	CurrentPort     *protocol.Port
	NextPort        *protocol.Port
	IsBreaked       bool
	HasNextPort     bool
	NeedSwitchGraph bool
}

func (p *fbpWorker) parseMessage(umsg mail.UserMessage) (msg *fbpMessage, err error) {

	session := umsg.Session()
	if session == nil {
		err = errors.New("payload session is nil")
		return
	}

	payload, ok := session.Payload().Interface().(*protocol.Payload)

	if !ok {
		err = errors.New("could not convert session payload to *protocol.Payload")
		return
	}

	currentGraphName := payload.GetCurrentGraph()
	currentGraph, exist := payload.GetGraph(currentGraphName)

	if !exist {
		err = fmt.Errorf("graph not exist: %s", currentGraphName)
		return
	}

	currentPort, err := currentGraph.CurrentPort()
	if err != nil {
		return
	}

	switchGraphName := GetSwitchGraphName(session)

	var nextGraph *protocol.Graph
	var nextPort *protocol.Port
	var hasNextPort, needSwitchGraph bool

	if len(switchGraphName) > 0 && switchGraphName != currentGraphName {
		switchGraph, exist := payload.GetGraph(switchGraphName)
		if !exist {
			err = fmt.Errorf("graph not exist: %s", switchGraphName)
			return
		}

		nextGraph = switchGraph
		nextPort, err = switchGraph.CurrentPort()

		if err != nil {
			return
		}

		needSwitchGraph = true

	} else {
		nextGraph = currentGraph
		nextPort, hasNextPort = currentGraph.NextPort()
	}

	retMsg := &fbpMessage{
		Session:         session,
		Payload:         payload,
		CurrentGraph:    currentGraph,
		NextGraph:       nextGraph,
		CurrentPort:     currentPort,
		NextPort:        nextPort,
		HasNextPort:     hasNextPort,
		IsBreaked:       IsSessionBreaked(session),
		NeedSwitchGraph: needSwitchGraph,
	}

	msg = retMsg

	return
}

func (p *fbpWorker) callHandler(router worker.HandlerRouter, session mail.Session) (err error) {

	if router == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			p.EscalateFailure(r, session)
			err = ErrExecuteHandlerPanic
			return
		}
	}()

	handler := router.Route(session)

	if handler == nil {
		return
	}

	err = handler(session)

	return
}

func (p *fbpWorker) process(umsg mail.UserMessage) {

	session := umsg.Session()
	if session == nil {
		p.EscalateFailure(errors.New("payload session is nil"), umsg)
		return
	}

	logrus.WithField("from", session.From()).
		WithField("to", session.To()).
		Debugln("enter FBP worker process")

	payload, ok := session.Payload().Interface().(*protocol.Payload)
	if !ok {
		p.EscalateFailure(errors.New("could not convert session payload to *protocol.Payload"), umsg)
		return
	}

	logrus.WithField("from", session.From()).
		WithField("to", session.To()).
		WithField("payload-id", payload.ID()).
		Traceln("payload found")

	errH := p.callHandler(p.opts.Router, session)
	if errH == nil &&
		payload.GetMessage().GetErr() != nil {
		errH = payload.GetMessage().GetErr()
	}

	if errH != nil && payload.GetCurrentGraph() != GraphNameOfError {
		logrus.WithError(errH).
			WithField("payload-id", payload.ID()).
			Errorln("execute handler error")
	}

	fbpMsg, err := p.parseMessage(umsg)

	if err != nil {
		p.EscalateFailure(err, umsg)
		return
	}

	if errH != nil {
		errGraph, exist := fbpMsg.Payload.GetGraph(GraphNameOfError)
		if !exist || errGraph == nil {
			p.EscalateFailure(
				fmt.Errorf("the payload did not have error graph, graph name: %s, handler error: %s", fbpMsg.CurrentGraph.GetName(), errH),
				umsg,
			)
			return
		}

		nextPort, errP := errGraph.CurrentPort()
		if errP != nil {
			p.EscalateFailure(
				fmt.Errorf("the payload get error graph's next port error, graph name: %s, handler error: %s, get error port error: %s", fbpMsg.CurrentGraph.GetName(), errH, errP),
				umsg,
			)
			return
		}

		if fbpMsg.CurrentGraph.GetName() != GraphNameOfError {
			fbpMsg.NextGraph = errGraph
			fbpMsg.NeedSwitchGraph = true
			fbpMsg.NextPort = nextPort
		}

		fbpMsg.Payload.Content().SetError(errH)
		fbpMsg.Session.WithError(errH)
	}

	// nothing todo while session breaked or did not have next port
	if fbpMsg.IsBreaked || !fbpMsg.HasNextPort {
		return
	}

	if fbpMsg.NeedSwitchGraph == false {
		fbpMsg.CurrentGraph.MoveForward()
	}

	fbpMsg.Session.WithFromTo(fbpMsg.CurrentPort.GetUrl(), fbpMsg.NextPort.GetUrl())

	SessionWithPort(fbpMsg.Session, fbpMsg.NextGraph.GetName(), fbpMsg.NextPort.GetUrl(), fbpMsg.NextPort.GetMetadata())

	fbpMsg.Payload.CurrentGraph = fbpMsg.NextGraph.Name

	err = p.opts.Postman.Post(message.NewUserMessage(fbpMsg.Session))

	if err != nil {
		p.EscalateFailure(
			fmt.Errorf("post message failure: %s, current graph: %s, seq: %d, next graph: %s, to url: %s",
				err,
				fbpMsg.CurrentGraph.GetName(),
				fbpMsg.CurrentGraph.GetSeq(),
				fbpMsg.NextGraph.GetName(),
				fbpMsg.NextPort.GetUrl(),
			),
			umsg,
		)
		return
	}

	return

}
