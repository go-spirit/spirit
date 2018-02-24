package mailbox

import (
	"github.com/go-spirit/spirit/mail"
)

type goroutineDispatcher int

func (goroutineDispatcher) Schedule(fn func()) {
	go fn()
}

func (d goroutineDispatcher) Throughput() int {
	return int(d)
}

func NewDefaultDispatcher(throughput int) mail.Dispatcher {
	return goroutineDispatcher(throughput)
}

type synchronizedDispatcher int

func (synchronizedDispatcher) Schedule(fn func()) {
	fn()
}

func (d synchronizedDispatcher) Throughput() int {
	return int(d)
}

func NewSynchronizedDispatcher(throughput int) mail.Dispatcher {
	return synchronizedDispatcher(throughput)
}
