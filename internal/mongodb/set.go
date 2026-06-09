package mongodb

import "strings"

type stringSet map[string]struct{}

func newStringSet(values ...string) stringSet {
	set := make(stringSet, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func (s stringSet) contains(value string) bool {
	_, ok := s[strings.ToLower(strings.TrimSpace(value))]
	return ok
}
