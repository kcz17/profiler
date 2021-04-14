package prioritystore

import "github.com/kcz17/profiler/priority"

type PriorityStore interface {
	Set(sessionID string, priority priority.Priority)
}
