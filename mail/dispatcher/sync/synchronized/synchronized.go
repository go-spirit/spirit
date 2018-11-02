package synchronized

import (
	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/mail/dispatcher"
)

func init() {
	dispatcher.RegisterDispatcher("sync", NewGoRoutineDispatcher)
}

func NewGoRoutineDispatcher(throughput int) (mail.Dispatcher, error) {
	return &synchronizedDispatcher{throughput}, nil
}

type synchronizedDispatcher struct {
	throughput int
}

func (p synchronizedDispatcher) Schedule(fn func()) {
	fn()
}

func (p synchronizedDispatcher) Throughput() int {
	return p.throughput
}
