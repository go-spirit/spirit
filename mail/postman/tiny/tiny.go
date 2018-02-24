package tiny

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-spirit/spirit/mail"
)

type TinyPostman struct {
	subscribers        map[string]mail.MailboxCallbackFunc
	inboundMiddlewares map[string]mail.InboundMiddlewareFunc

	opts mail.PostmanOptions

	stopC chan bool
}

func init() {
	mail.RegisterPostman("tiny", NewTinyPostman)
}

func NewTinyPostman(opts ...mail.PostmanOption) (postman mail.Postman, err error) {

	pm := &TinyPostman{
		subscribers:        make(map[string]mail.MailboxCallbackFunc),
		inboundMiddlewares: make(map[string]mail.InboundMiddlewareFunc),
		stopC:              make(chan bool),
	}

	for _, o := range opts {
		o(&pm.opts)
	}

	postman = pm

	return
}

func (p *TinyPostman) Registry() mail.Registry {
	return p.opts.Registry
}

func (p *TinyPostman) SubscribeReceived(url string, fn mail.MailboxCallbackFunc) {
	parts := strings.Split(url, "?")
	p.subscribers[parts[0]] = fn
	return
}

func (p *TinyPostman) Start() error {
	boxes := p.Registry().AllMailBoxes()

	for i := 0; i < len(boxes); i++ {
		boxes[i].Start()
	}

	return nil
}

func (p *TinyPostman) Post(m mail.Message) (err error) {
	switch v := m.(type) {
	case mail.UserMessage:
		{
			session := v.Session()
			if session == nil {
				err = errors.New("session is nil")
				return
			}

			dstBox, exist := p.opts.Registry.GetMailBox(session.To())

			if !exist {
				err = fmt.Errorf("the box not find of url: %s", session.To())
				return
			}

			dstBox.PostUserMessage(m)

			return
		}
	}

	return
}
