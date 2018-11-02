package dispatcher

import (
	"fmt"
	"github.com/go-spirit/spirit/mail"
)

type NewDispatcherFunc func(throughput int) (mail.Dispatcher, error)

var (
	dispatchers map[string]NewDispatcherFunc = make(map[string]NewDispatcherFunc)
)

func RegisterDispatcher(name string, fn NewDispatcherFunc) {
	if len(name) == 0 {
		panic("dispatcher name is empty")
	}

	if fn == nil {
		panic("dispatcher fn is nil")
	}

	_, exist := dispatchers[name]

	if exist {
		panic(fmt.Sprintf("dispatcher already registered: %s", name))
	}

	dispatchers[name] = fn
}

func NewDispatcher(name string, throughput int) (dispatcher mail.Dispatcher, err error) {
	fn, exist := dispatchers[name]

	if !exist {
		err = fmt.Errorf("dispatcher not registered '%s'", name)
		return
	}

	dispatcher, err = fn(throughput)

	return
}
