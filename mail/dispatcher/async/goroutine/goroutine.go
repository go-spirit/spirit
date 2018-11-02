package goroutine

import (
	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/mail/dispatcher"
)

func init() {
	dispatcher.RegisterDispatcher("goroutine", NewGoRoutineDispatcher)
}

func NewGoRoutineDispatcher(throughput int) (mail.Dispatcher, error) {
	return &goroutineDispatcher{throughput}, nil
}

type goroutineDispatcher struct {
	throughput int
}

func (p goroutineDispatcher) Schedule(fn func()) {
	go fn()
}

func (p goroutineDispatcher) Throughput() int {
	return p.throughput
}
