package codec

import (
	"encoding/json"
)

type Json struct {
}

func NewJsonCodec() (Codec, error) {
	return &Json{}, nil
}

func (p *Json) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (p *Json) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
