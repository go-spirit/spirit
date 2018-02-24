package mail

import (
	"fmt"
)

type Registry interface {
	Register(Mailbox) error
	GetMailBox(string) (Mailbox, bool)
	AllMailBoxes() []Mailbox
}

type NewRegistryFunc func() (Registry, error)

var (
	registries map[string]NewRegistryFunc = make(map[string]NewRegistryFunc)
)

func RegisterRegistry(name string, fn NewRegistryFunc) {

	if len(name) == 0 {
		panic("registry name is empty")
	}

	if fn == nil {
		panic("registry fn is nil")
	}

	_, exist := registries[name]

	if exist {
		panic(fmt.Sprintf("registry already registered: %s", name))
	}

	registries[name] = fn
}

func NewRegistry(name string) (room Registry, err error) {
	fn, exist := registries[name]

	if !exist {
		err = fmt.Errorf("registry not exist '%s'", name)
		return
	}

	room, err = fn()

	return
}
