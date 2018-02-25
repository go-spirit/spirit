package fbp

import (
	"github.com/go-spirit/spirit/mail"
	"github.com/sirupsen/logrus"
)

func (p *fbpWorker) InvokeSystemMessage(message interface{}) {

}

func (p *fbpWorker) InvokeUserMessage(message interface{}) {
	umsg, ok := message.(mail.UserMessage)
	if !ok {
		return
	}

	p.process(umsg)
}

func (p *fbpWorker) EscalateFailure(reason interface{}, message interface{}) {
	logrus.WithField("message", message).Errorln(reason)
}
