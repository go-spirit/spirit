package tiny

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-spirit/spirit/mail"
)

type TinyRegistry struct {
	loc   sync.Mutex
	boxes map[string]mail.Mailbox
}

func init() {
	mail.RegisterRegistry("tiny", NewTinyRegistry)
}

func NewTinyRegistry() (reg mail.Registry, err error) {
	reg = &TinyRegistry{
		boxes: make(map[string]mail.Mailbox),
	}
	return
}

func (p *TinyRegistry) Register(box mail.Mailbox) (err error) {
	if box == nil {
		err = errors.New("mailbox is nil")
		return
	}

	if len(box.Url()) == 0 {
		err = errors.New("url is empty")
		return
	}

	p.loc.Lock()
	defer p.loc.Unlock()

	_, exist := p.boxes[box.Url()]
	if exist {
		err = fmt.Errorf("url already exist: %s", box.Url)
		return
	}

	p.boxes[box.Url()] = box

	return
}

func (p *TinyRegistry) GetMailBox(strUrl string) (box mail.Mailbox, exist bool) {

	parts := strings.Split(strUrl, "?")

	box, exist = p.boxes[parts[0]]
	return
}

func (p *TinyRegistry) AllMailBoxes() []mail.Mailbox {
	var all []mail.Mailbox

	for _, box := range p.boxes {
		all = append(all, box)
	}

	return all
}
