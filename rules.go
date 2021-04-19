package main

import (
	"github.com/kcz17/profiler/priority"
	"strings"
)

type MatchableMethod struct {
	ShouldMatchAll bool
	// Method must be set if ShouldMatchAll is false. If ShouldMatchAll is true,
	// Method is ignored.
	Method string
}

func (m MatchableMethod) IsMatch(method string) bool {
	return m.ShouldMatchAll || strings.ToUpper(method) == strings.ToUpper(m.Method)
}

type Rule struct {
	Description string
	Method      MatchableMethod
	Path        string
	Occurrences int
	Result      priority.Priority
}

func (r Rule) IsMatch(method, path string) bool {
	return r.Method.IsMatch(method) && r.Path == path
}

// OrderedRules is a slice of rules ordered by highest priority rule descending.
type OrderedRules []Rule
