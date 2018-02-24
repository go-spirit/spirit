package worker

import (
	"fmt"

	"github.com/go-spirit/spirit/mail"
)

type Worker interface {
	Init(...WorkerOption)

	mail.MessageInvoker
}

type WorkerOptions struct {
	Postman mail.Postman
	Handler HandlerFunc
}

type WorkerOption func(*WorkerOptions)

func Postman(man mail.Postman) WorkerOption {
	return func(o *WorkerOptions) {
		o.Postman = man
	}
}

func Handler(h HandlerFunc) WorkerOption {
	return func(o *WorkerOptions) {
		o.Handler = h
	}
}

type NewWorkerFunc func(...WorkerOption) (Worker, error)

var (
	workers map[string]NewWorkerFunc = make(map[string]NewWorkerFunc)
)

func RegisterWorker(name string, fn NewWorkerFunc) {
	if len(name) == 0 {
		panic("worker name is empty")
	}

	if fn == nil {
		panic("worker fn is nil")
	}

	_, exist := workers[name]

	if exist {
		panic(fmt.Sprintf("worker already register: %s", name))
	}

	workers[name] = fn
}

func New(name string, opts ...WorkerOption) (act Worker, err error) {
	fn, exist := workers[name]

	if !exist {
		err = fmt.Errorf("worker not exist '%s'", name)
		return
	}

	act, err = fn(opts...)

	return
}
