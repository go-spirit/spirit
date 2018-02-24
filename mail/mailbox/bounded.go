package mailbox

import (
	rbqueue "github.com/Workiva/go-datastructures/queue"
	"github.com/go-spirit/spirit/internal/queue/mpsc"
	"github.com/go-spirit/spirit/mail"
)

type boundedMailboxQueue struct {
	userMailbox *rbqueue.RingBuffer
	dropping    bool

	opts mail.MailboxOptions
}

func (q *boundedMailboxQueue) Push(m interface{}) {
	if q.dropping {
		if q.userMailbox.Len() > 0 && q.userMailbox.Cap()-1 == q.userMailbox.Len() {
			q.userMailbox.Get()
		}
	}
	q.userMailbox.Put(m)
}

func (q *boundedMailboxQueue) Pop() interface{} {
	if q.userMailbox.Len() > 0 {
		m, _ := q.userMailbox.Get()
		return m
	}
	return nil
}

func init() {
	mail.RegisterMailbox("bounded", NewBoundedMailbox)
}

func NewBoundedMailbox(opts ...mail.MailboxOption) (mail.Mailbox, error) {

	mbOpts := &mail.MailboxOptions{
		Dispatcher: defaultDispatcher,
	}

	for _, o := range opts {
		o(mbOpts)
	}

	q := &boundedMailboxQueue{
		userMailbox: rbqueue.NewRingBuffer(uint64(mbOpts.MailboxSize)),
		dropping:    mbOpts.DroppingOnFull,
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
