package mailbox

import (
	"runtime"
	"sync/atomic"

	"github.com/go-spirit/spirit/internal/queue/mpsc"
	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/message"
	"github.com/gogap/errors"
)

var (
	mailboxErrNS = "MAILBOX"
)

var (
	ErrMailboxRecover = errors.TN(mailboxErrNS, 10000, "Recovering, reason: {{.reason}}")
)

// Producer is a function which creates a new mailbox
type Producer func(invoker mail.MessageInvoker, dispatcher mail.Dispatcher) mail.Inbound

const (
	idle int32 = iota
	running
)

type defaultMailbox struct {
	userMailbox     queue
	systemMailbox   *mpsc.Queue
	schedulerStatus int32
	userMessages    int32
	sysMessages     int32
	invoker         mail.MessageInvoker
	dispatcher      mail.Dispatcher
	suspended       int32
	mailboxStats    []mail.Statistics
	url             string
}

func (p *defaultMailbox) Url() string {
	return p.url
}

func (m *defaultMailbox) PostUserMessage(message interface{}) {
	for _, ms := range m.mailboxStats {
		ms.MessagePosted(message)
	}
	m.userMailbox.Push(message)
	atomic.AddInt32(&m.userMessages, 1)
	m.schedule()
}

func (m *defaultMailbox) PostSystemMessage(message interface{}) {
	for _, ms := range m.mailboxStats {
		ms.MessagePosted(message)
	}
	m.systemMailbox.Push(message)
	atomic.AddInt32(&m.sysMessages, 1)
	m.schedule()
}

func (m *defaultMailbox) schedule() {
	if atomic.CompareAndSwapInt32(&m.schedulerStatus, idle, running) {
		m.dispatcher.Schedule(m.processMessages)
	}
}

func (m *defaultMailbox) processMessages() {

process:
	m.run()

	// set mailbox to idle
	atomic.StoreInt32(&m.schedulerStatus, idle)
	sys := atomic.LoadInt32(&m.sysMessages)
	user := atomic.LoadInt32(&m.userMessages)
	// check if there are still messages to process (sent after the message loop ended)
	if sys > 0 || (atomic.LoadInt32(&m.suspended) == 0 && user > 0) {
		// try setting the mailbox back to running
		if atomic.CompareAndSwapInt32(&m.schedulerStatus, idle, running) {
			//	fmt.Printf("looping %v %v %v\n", sys, user, m.suspended)
			goto process
		}
	}

	for _, ms := range m.mailboxStats {
		ms.MailboxEmpty()
	}
}

func (m *defaultMailbox) run() {
	var msg interface{}

	defer func() {
		if r := recover(); r != nil {
			m.invoker.EscalateFailure(ErrMailboxRecover.New(errors.Params{"reason": r}), msg)
		}
	}()

	i, t := 0, m.dispatcher.Throughput()
	for {
		if i > t {
			i = 0
			runtime.Gosched()
		}

		i++

		// keep processing system messages until queue is empty
		if msg = m.systemMailbox.Pop(); msg != nil {
			atomic.AddInt32(&m.sysMessages, -1)
			switch msg.(type) {
			case *message.SuspendMailbox:
				atomic.StoreInt32(&m.suspended, 1)
			case *message.ResumeMailbox:
				atomic.StoreInt32(&m.suspended, 0)
			default:
				m.invoker.InvokeSystemMessage(msg)
			}
			for _, ms := range m.mailboxStats {
				ms.MessageReceived(msg)
			}
			continue
		}

		// didn't process a system message, so break until we are resumed
		if atomic.LoadInt32(&m.suspended) == 1 {
			return
		}

		if msg = m.userMailbox.Pop(); msg != nil {
			atomic.AddInt32(&m.userMessages, -1)
			m.invoker.InvokeUserMessage(msg)
			for _, ms := range m.mailboxStats {
				ms.MessageReceived(msg)
			}
		} else {
			return
		}
	}

}

func (m *defaultMailbox) Start() {
	for _, ms := range m.mailboxStats {
		ms.MailboxStarted()
	}
}
