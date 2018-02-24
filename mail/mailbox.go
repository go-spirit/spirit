package mail

import (
	"fmt"
)

type Mailboxes struct {
	Url    string
	Inbox  Mailbox
	Outbox Mailbox
}

type Dispatcher interface {
	Schedule(fn func())
	Throughput() int
}

type Mailbox interface {
	Url() string
	Start()
	// Push(Message, InboundMiddlewareFunc, MailboxCallbackFunc) error
	// Pop() (Message, bool)

	Inbound
}

// MessageInvoker is the interface used by a mailbox to forward messages for processing
type MessageInvoker interface {
	InvokeSystemMessage(interface{})
	InvokeUserMessage(interface{})
	EscalateFailure(reason interface{}, message interface{})
}

type Statistics interface {
	MailboxStarted()
	MessagePosted(message interface{})
	MessageReceived(message interface{})
	MailboxEmpty()
}

type MailboxOptions struct {
	Url            string
	MailboxSize    int
	DroppingOnFull bool
	Statistics     []Statistics
	Invoker        MessageInvoker
	Dispatcher     Dispatcher
}

type MailboxOption func(*MailboxOptions)

type NewMailboxFunc func(...MailboxOption) (Mailbox, error)

func MailboxUrl(url string) MailboxOption {
	return func(o *MailboxOptions) {
		o.Url = url
	}
}

func MailboxSize(size int) MailboxOption {
	return func(o *MailboxOptions) {
		o.MailboxSize = size
	}
}

func MailboxDroppingOnFull(dropping bool) MailboxOption {
	return func(o *MailboxOptions) {
		o.DroppingOnFull = dropping
	}
}

func MailboxStatistics(s ...Statistics) MailboxOption {
	return func(o *MailboxOptions) {
		o.Statistics = s
	}
}

func MailboxMessageInvoker(invoker MessageInvoker) MailboxOption {
	return func(o *MailboxOptions) {
		o.Invoker = invoker
	}
}

func MailboxDispatcher(d Dispatcher) MailboxOption {
	return func(o *MailboxOptions) {
		o.Dispatcher = d
	}
}

var (
	mailboxes = make(map[string]NewMailboxFunc)
)

func RegisterMailbox(name string, fn NewMailboxFunc) {
	if len(name) == 0 {
		panic("mailbox name is empty")
	}

	if fn == nil {
		panic("mailbox fn is nil")
	}

	_, exist := mailboxes[name]

	if exist {
		panic(fmt.Sprintf("mailbox already registered: %s", name))
	}

	mailboxes[name] = fn
}

func NewMailbox(name string, opts ...MailboxOption) (box Mailbox, err error) {
	fn, exist := mailboxes[name]

	if !exist {
		err = fmt.Errorf("mailbox driver not exist '%s'", name)
		return
	}

	box, err = fn(opts...)

	return
}
