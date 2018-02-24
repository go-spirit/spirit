package protocol

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	gogaperrors "github.com/gogap/errors"

	"github.com/go-spirit/spirit/codec"
	"github.com/go-spirit/spirit/mail"

	proto "github.com/golang/protobuf/proto"
)

func (p *Error) Error() string {
	return fmt.Sprintf("[%s:%d] %s", p.Namespace, p.Code, p.Description)
}

func (p *Error) WithContext(key, value string) {
	if p.Context == nil {
		p.Context = make(map[string]string)
	}
	p.Context[key] = value
}

func Unmarshal(data []byte, v proto.Message) error {
	return proto.Unmarshal(data, v)
}

func (p *Payload) ToBytes() ([]byte, error) {
	return proto.Marshal(p)
}

func (p *Graph) MoveForward() (err error) {
	seq := p.GetSeq()
	ports := p.GetPorts()

	if len(ports) == 0 {
		err = errors.New("payload graph's ports is empty")
		return
	}

	if seq+1 < 0 || int(seq+1) >= len(ports) {
		err = errors.New("bad graph seq while move forward")
		return
	}

	p.Seq += 1

	return
}

func (p *Graph) CurrentPort() (port *Port, err error) {
	seq := p.GetSeq()
	ports := p.GetPorts()

	if len(ports) == 0 {
		err = errors.New("payload graph's ports is empty")
		return
	}

	if seq < 0 || int(seq) >= len(ports) {
		err = errors.New("bad graph seq while get current port")
		return
	}

	port = ports[seq]

	return
}

func (p *Graph) PrevPort() (port *Port, err error) {
	seq := p.GetSeq()
	ports := p.GetPorts()

	if len(ports) == 0 {
		err = errors.New("payload graph's ports is empty")
		return
	}

	if seq-1 < 0 || int(seq-1) >= len(ports) {
		err = errors.New("bad graph seq while get prev port")
		return
	}

	port = ports[seq-1]

	return
}

func (p *Graph) NextPort() (port *Port, has bool) {
	seq := p.GetSeq()
	ports := p.GetPorts()

	if len(ports) == 0 {
		return nil, false
	}

	if seq+1 < 0 || int(seq+1) >= len(ports) {
		return nil, false
	}

	port = ports[seq+1]

	return port, true
}

func (p *Port) GetUrlQuery() (values url.Values, err error) {

	u, err := url.Parse(p.GetUrl())
	if err != nil {
		return
	}

	values = u.Query()

	return
}

func (p *Port) TrimUrlQuery() string {
	parts := strings.Split(p.GetUrl(), "?")
	return parts[0]
}

func (p *Message) ContentType() (ct string, exist bool) {
	headers := p.GetHeader()
	if headers == nil {
		return "", false
	}

	ct, exist = headers["content-type"]

	return
}

func (p *Message) Unmarshal(v interface{}) (err error) {
	if p == nil {
		err = errors.New("message is nil")
		return
	}

	ct, exist := p.ContentType()

	if !exist {
		err = errors.New("content type is empty")
		return
	}

	decoder := codec.Select(ct)

	if decoder == nil {
		err = fmt.Errorf("not codec for content type: %s", ct)
		return
	}

	return decoder.Unmashal(p.GetBody(), v)
}

func (p *Message) Marshal() (data []byte, err error) {

	if p == nil {
		return
	}

	ct, exist := p.ContentType()

	if !exist {
		err = errors.New("content type is empty")
		return
	}

	encoder := codec.Select(ct)

	if encoder == nil {
		err = fmt.Errorf("not codec for content type: %s", ct)
		return
	}

	return encoder.Marshal(p)
}

func (p *Message) SetBody(v interface{}) (err error) {

	if v == nil {
		return
	}

	switch val := v.(type) {
	case []byte:
		{
			p.Body = val
		}
	default:

		ct, exist := p.ContentType()

		if !exist {
			err = errors.New("content type is empty")
			return
		}

		encoder := codec.Select(ct)

		if encoder == nil {
			err = fmt.Errorf("not codec for content type: %s", ct)
			return
		}

		var data []byte
		data, err = encoder.Marshal(v)

		if err != nil {
			return
		}

		p.Body = data
	}

	return
}

func (p *Message) Copy() mail.Content {

	header := map[string]string{}

	for k, v := range p.GetHeader() {
		header[k] = v
	}

	var body []byte

	copy(body, p.GetBody())

	n := &Message{
		Id:     p.GetId(),
		Header: header,
		Body:   body,
	}

	return n
}

func (p *Message) SetError(err error) {
	if p == nil {
		return
	}

	switch e := err.(type) {
	case *Error:
		{
			p.Error = e

			return
		}
	case gogaperrors.ErrCode:
		{
			newErr := &Error{
				Namespace:   e.Namespace(),
				Code:        int64(e.Code()),
				Description: e.Error(),
				Stack:       e.StackTrace(),
				Context:     make(map[string]string),
			}

			for k, v := range e.Context() {
				newErr.Context[k] = fmt.Sprintf("%v", v)
			}

			p.Error = newErr

			return
		}
	default:
		{
			newErr := &Error{
				Namespace:   "SPIRIT",
				Code:        0xdeaddead,
				Description: e.Error(),
			}

			p.Error = newErr

			return
		}
	}

	return
}

func (p *Payload) Content() mail.Content {
	return p.GetMessage()
}
