package mns

import (
	"errors"
	"fmt"
	"sync"

	"github.com/go-spirit/spirit/component"
	"github.com/go-spirit/spirit/doc"
	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/message"
	"github.com/go-spirit/spirit/worker"
	"github.com/go-spirit/spirit/worker/fbp"
	"github.com/go-spirit/spirit/worker/fbp/protocol"
	"github.com/gogap/ali_mns"
	"github.com/sirupsen/logrus"
)

type mnsQueue struct {
	Name            string
	Endpoint        string
	AccessKeyId     string
	AccessKeySecret string
	Queue           ali_mns.AliMNSQueue
}

type MNSComponent struct {
	opts component.Options

	queues []mnsQueue

	endpoint        string
	accessKeyId     string
	accessKeySecret string

	respChan chan ali_mns.BatchMessageReceiveResponse
	errChan  chan error

	stopC chan bool
}

func init() {
	component.RegisterComponent("mns", NewMNSComponent)
	doc.RegisterDocumenter("mns", &MNSComponent{})
}

func NewMNSComponent(opts ...component.Option) (comp component.Component, err error) {
	mnsComp := &MNSComponent{
		stopC:    make(chan bool),
		respChan: make(chan ali_mns.BatchMessageReceiveResponse, 30),
		errChan:  make(chan error, 30),
	}

	err = mnsComp.init(opts...)
	if err != nil {
		return
	}

	comp = mnsComp

	return
}

func (p *MNSComponent) init(opts ...component.Option) (err error) {

	for _, o := range opts {
		o(&p.opts)
	}

	if p.opts.Config == nil {
		err = errors.New("mns component config is nil")
		return
	}

	akId := p.opts.Config.GetString("access-key-id")
	akSecret := p.opts.Config.GetString("access-key-secret")
	endpoint := p.opts.Config.GetString("endpoint")

	p.accessKeyId = akId
	p.accessKeySecret = akSecret
	p.endpoint = endpoint

	queuesConf := p.opts.Config.GetConfig("queues")

	if queuesConf == nil {
		return
	}

	qNames := queuesConf.Keys()

	var mnsQueues []mnsQueue

	for _, name := range qNames {
		endpoint := queuesConf.GetString("endpoint", endpoint)
		qAkId := queuesConf.GetString("access-key-id", akId)
		qAkSecret := queuesConf.GetString("access-key-secret", akSecret)

		qClient := ali_mns.NewAliMNSClient(endpoint, qAkId, qAkSecret)
		aliQueue := ali_mns.NewMNSQueue(name, qClient)

		q := mnsQueue{
			Name:            name,
			Endpoint:        endpoint,
			AccessKeyId:     qAkId,
			AccessKeySecret: qAkSecret,
			Queue:           aliQueue,
		}

		mnsQueues = append(mnsQueues, q)
	}

	p.queues = mnsQueues

	return
}

func (p *MNSComponent) Start() error {

	for _, q := range p.queues {
		mgr := ali_mns.NewMNSQueueManager(q.AccessKeyId, q.AccessKeySecret)
		_, err := mgr.GetQueueAttributes(q.Endpoint, q.Name)
		if err != nil {
			return err
		}
	}

	for _, q := range p.queues {
		go q.Queue.BatchReceiveMessage(p.respChan, p.errChan, 16, 30)
	}

	go p.receiveMessage()

	return nil
}

func (p *MNSComponent) postMessage(resp ali_mns.MessageReceiveResponse) (err error) {

	payload := &protocol.Payload{}
	err = protocol.Unmarshal([]byte(resp.MessageBody), payload)

	if err != nil {
		return
	}

	graph, exist := payload.GetGraph(payload.GetCurrentGraph())
	if !exist {
		err = fmt.Errorf("could not get graph of %s in MNSComponent.postMessage", payload.GetCurrentGraph())
		return
	}

	graph.MoveForward()

	port, err := graph.CurrentPort()

	if err != nil {
		return
	}

	fromUrl := ""
	prePort, preErr := graph.PrevPort()
	if preErr == nil {
		fromUrl = prePort.GetUrl()
	}

	session := mail.NewSession()

	session.WithPayload(payload)
	session.WithFromTo(fromUrl, port.GetUrl())

	fbp.SessionWithPort(session, port.GetUrl(), false, port.GetMetadata())

	err = p.opts.Postman.Post(
		message.NewUserMessage(session),
	)

	if err != nil {
		return
	}

	return
}

func (p *MNSComponent) receiveMessage() {
	for {
		select {
		case resp, ok := <-p.respChan:
			{
				if !ok {
					break
				}

				for _, m := range resp.Messages {
					err := p.postMessage(m)
					if err != nil {
						logrus.WithField("component", "mns").WithError(err).Errorln("post received mns message failured")
					}
				}
			}
		case err, ok := <-p.errChan:
			{
				if !ok {
					break
				}

				if !ali_mns.ERR_MNS_MESSAGE_NOT_EXIST.IsEqual(err) {
					logrus.WithField("component", "mns").WithError(err).Errorln("receive mns message failure")
				}
			}
		case <-p.stopC:
			{
				p.stopC <- true
				break
			}
		}
	}
}

func (p *MNSComponent) Stop() error {

	if len(p.queues) == 0 {
		return nil
	}

	logrus.WithField("component", "mns").WithField("queue-count", len(p.queues)).Infoln("stopping...")

	wg := &sync.WaitGroup{}
	wg.Add(len(p.queues))

	for _, q := range p.queues {
		go func(queue mnsQueue) {
			defer wg.Done()
			logrus.WithField("component", "mns").WithField("queue", queue.Name).Infoln("stopping...")
			queue.Queue.Stop()
			logrus.WithField("component", "mns").WithField("queue", queue.Name).Infoln("stopped.")
		}(q)
	}

	wg.Wait()

	p.stopC <- true
	<-p.stopC

	close(p.errChan)
	close(p.respChan)
	close(p.stopC)

	logrus.WithField("component", "mns").Infoln("all queues listener stopped.")

	return nil
}

// It is a send out func
func (p *MNSComponent) sendMessage(session mail.Session) (err error) {

	logrus.WithField("component", "MNSComponent").WithField("To", session.To()).Debugln("send message")

	fbp.BreakSession(session)

	port := fbp.GetSessionPort(session)

	if port == nil {
		err = errors.New("port info not exist")
		return
	}

	queueName := session.Query("queue")

	if len(queueName) == 0 {
		err = fmt.Errorf("queue name is empty")
		return
	}

	endpoint, exist := port.Metadata["endpoint"]

	if !exist {
		endpoint = p.endpoint
	}

	akId, exist := port.Metadata["access-key-id"]
	if !exist {
		akId = p.accessKeyId
	}

	akSecret, exist := port.Metadata["access-key-secret"]

	if !exist {
		akSecret = p.accessKeySecret
	}

	if len(endpoint) == 0 || len(akId) == 0 || len(akSecret) == 0 {
		err = fmt.Errorf("error mns send params in msn component, port to url: %s", port.Url)
		return
	}

	client := ali_mns.NewAliMNSClient(endpoint, akId, akSecret)
	queue := ali_mns.NewMNSQueue(queueName, client)

	payload, ok := session.Payload().Interface().(*protocol.Payload)
	if !ok {
		err = errors.New("could not convert session payload to *protocol.Payload")
		return
	}

	data, err := payload.ToBytes()

	if err != nil {
		return
	}

	req := ali_mns.MessageSendRequest{
		MessageBody: data,
		Priority:    8,
	}

	_, err = queue.SendMessage(req)

	if err != nil {
		return
	}

	return
}

func (p *MNSComponent) Route(mail.Session) worker.HandlerFunc {
	return p.sendMessage
}

func (p *MNSComponent) Document() doc.Document {

	document := doc.Document{
		Title:       "MNS Sender And Receiver",
		Description: "MNS is aliyun message service, we could receive queue message from msn and send message to msn queue",
	}

	return document
}
