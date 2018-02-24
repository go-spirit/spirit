package tags

import (
	"sort"
	"strings"
)

type Tag string

type Tags []string

func (p Tag) Equals(t Tag) bool {
	return p == t
}

func (p Tags) Equals(tags Tags) bool {
	if len(p) != len(tags) {
		return false
	}

	t := map[string]bool{}

	for _, tag := range p {
		t[tag] = true
	}

	for _, tag := range tags {
		if t[tag] == false {
			return false
		}
	}

	return true
}

func (p Tags) IsEmpty() bool {
	return len(p) == 0
}

func (p Tags) Unique() string {

	str := make([]string, len(p), len(p))

	for i := 0; i < len(p); i++ {
		str[i] = string(p[i])
	}

	sort.Strings(str)
	return strings.Join(str, "-")
}
