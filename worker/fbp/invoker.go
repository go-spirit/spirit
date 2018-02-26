package fbp

import (
	"github.com/go-spirit/spirit/mail"
	"github.com/sirupsen/logrus"
	"runtime"
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

	var buf = make([]byte, 2048)
	runtime.Stack(buf, false)

	logrus.WithField("message", message).Errorln(reason)
	logrus.Debugln(string(buf))
}
