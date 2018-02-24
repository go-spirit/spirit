package mailbox

import (
	"github.com/go-spirit/spirit/internal/queue/goring"
	"github.com/go-spirit/spirit/internal/queue/mpsc"
	"github.com/go-spirit/spirit/mail"
)

type unboundedMailboxQueue struct {
	userMailbox *goring.Queue
}

func (q *unboundedMailboxQueue) Push(m interface{}) {
	q.userMailbox.Push(m)
}

func (q *unboundedMailboxQueue) Pop() interface{} {
	m, o := q.userMailbox.Pop()
	if o {
		return m
	}
	return nil
}

func init() {
	mail.RegisterMailbox("unbounded", NewUnboundedMailbox)
}

func NewUnboundedMailbox(opts ...mail.MailboxOption) (mail.Mailbox, error) {

	mbOpts := &mail.MailboxOptions{
		Dispatcher: defaultDispatcher,
	}

	for _, o := range opts {
		o(mbOpts)
	}

	q := &unboundedMailboxQueue{
		userMailbox: goring.New(10),
	}

	return &defaultMailbox{
		systemMailbox: mpsc.New(),
		userMailbox:   q,
		invoker:       mbOpts.Invoker,
		mailboxStats:  mbOpts.Statistics,
		dispatcher:    mbOpts.Dispatcher,
		url:           mbOpts.Url,
	}, nil
}
