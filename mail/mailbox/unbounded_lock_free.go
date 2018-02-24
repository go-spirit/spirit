package mailbox

import (
	"github.com/go-spirit/spirit/internal/queue/mpsc"
	"github.com/go-spirit/spirit/mail"
)

func init() {
	mail.RegisterMailbox("unbounded_lock_free", NewUnboundedLockfreeMailbox)
}

func NewUnboundedLockfreeMailbox(opts ...mail.MailboxOption) (mail.Mailbox, error) {

	mbOpts := &mail.MailboxOptions{
		Dispatcher: defaultDispatcher,
	}

	for _, o := range opts {
		o(mbOpts)
	}

	return &defaultMailbox{
		systemMailbox: mpsc.New(),
		userMailbox:   mpsc.New(),
		invoker:       mbOpts.Invoker,
		mailboxStats:  mbOpts.Statistics,
		dispatcher:    mbOpts.Dispatcher,
		url:           mbOpts.Url,
	}, nil
}
