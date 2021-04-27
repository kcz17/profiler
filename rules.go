package main

import (
	"fmt"
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

func (rules OrderedRules) String() string {
	var sb strings.Builder

	for i, rule := range rules {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i, rule.Description))
		if rule.Method.ShouldMatchAll {
			sb.WriteString("\tMethod: * \n")
		} else {
			sb.WriteString(fmt.Sprintf("\tMethod: %s \n", rule.Method.Method))
		}
		sb.WriteString(fmt.Sprintf("\tPath: %s \n", rule.Path))
		sb.WriteString(fmt.Sprintf("\tOccurrences needed: %d \n", rule.Occurrences))
		sb.WriteString(fmt.Sprintf("\tResulting priority: %s \n", rule.Result.String()))
	}

	return sb.String()
}
