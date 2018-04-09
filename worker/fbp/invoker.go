package fbp

import (
	"bytes"
	"runtime"

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

	var buf = make([]byte, 4096)
	runtime.Stack(buf, false)

	logrus.WithField("message", message).WithField("stacktrace", string(bytes.Trim(buf, "\x00"))).Errorln(reason)
}
