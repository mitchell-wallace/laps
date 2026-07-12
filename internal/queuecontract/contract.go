// Package queuecontract defines the stable queue-state exit codes shared by
// command behavior and agent-facing documentation checks.
package queuecontract

const (
	ExitRun      = 0
	ExitHeld     = 10
	ExitEmpty    = 11
	ExitComplete = 12
)
