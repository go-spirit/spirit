package codec

import (
	"fmt"
)

type Codec interface {
	Unmashal([]byte, interface{}) error
	Marshal(interface{}) ([]byte, error)
}

type NewCodecFunc func() (Codec, error)

var (
	codecs map[string]NewCodecFunc = make(map[string]NewCodecFunc)
)

var (
	defaultCodecs map[string]Codec = make(map[string]Codec)
)

func init() {
	RegisterCodec("application/json", NewJsonCodec)

	jsonCodec, _ := NewCodec("application/json")
	defaultCodecs["application/json"] = jsonCodec
}

func Select(contentType string) Codec {
	return defaultCodecs[contentType]
}

func RegisterCodec(name string, fn NewCodecFunc) {
	if len(name) == 0 {
		panic("codec name is empty")
	}

	if fn == nil {
		panic("codec fn is nil")
	}

	_, exist := codecs[name]

	if exist {
		panic(fmt.Sprintf("codec already registered: %s", name))
	}

	codecs[name] = fn
}

func NewCodec(name string) (c Codec, err error) {
	fn, exist := codecs[name]

	if !exist {
		err = fmt.Errorf("codec not exist '%s'", name)
		return
	}

	c, err = fn()

	return
}
