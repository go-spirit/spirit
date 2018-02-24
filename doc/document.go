package doc

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Field struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Length      int    `json:"length,omitempty"`
	Example     string `json:"example,omitempty"`
	Default     string `json:"default,omitempty"`
	Range       string `json:"range,omitempty"`
	IsArray     bool   `json:"is_array,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
	Description string `json:"description,omitempty"`

	Fields []*Field `json:"fields,omitempty"`
}

type MaintainLog struct {
	Maintainer string `json:"maintainer"`
	Content    string `json:"content"`
	UpdateAt   string `json:"update_at"`
}

type Document struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Contacts    []Contact `json:"contacts"`

	Input  []*Field `json:"intput"`
	Output []*Field `json:"output"`

	ConfigTemplate string `json:"config_template"`

	MaintainLogs []MaintainLog `json:"maintain_logs"`

	Tags []string `json:"tags"`
}

func (p *Document) JSON() (string, error) {
	if p == nil {
		return "", nil
	}

	data, err := json.MarshalIndent(p, "", "    ")

	return string(data), err
}

func (p *Document) Markdown() string {
	return ""
}

type Documenter interface {
	Document() Document
}

var (
	documenters = make(map[string]Documenter)
)

func GetDocumenter(name string) (Documenter, bool) {
	doc, exist := documenters[name]
	return doc, exist
}

func RegisterDocumenter(name string, documenter Documenter) (err error) {

	if len(name) == 0 {
		err = errors.New("document name is empty")
		return
	}

	if documenter == nil {
		err = errors.New("documenter is nil")
		return
	}

	_, exist := documenters[name]
	if exist {
		err = fmt.Errorf("documenter of %s already exist", name)
		return
	}

	documenters[name] = documenter

	return
}
