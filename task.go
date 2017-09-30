package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrConcurrentExecution = errors.New("cannot allow concurrent executions")
	ErrTimeoutExceeded     = errors.New("timeout exceeded")
)

// MSDuration is a convenience type representing milliseconds
type MSDuration time.Duration

func (m MSDuration) MarshalJSON() ([]byte, error) {
	ms := float64(m) / float64(time.Millisecond)
	return json.Marshal(ms)
}

func (m *MSDuration) UnmarshalJSON(b []byte) error {
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("couldn't unmarshal MSDuration: %s", err)
	}
	*m = MSDuration(f * float64(time.Millisecond))
	return nil
}

// TaskRequest is a request to run a task.
type TaskRequest struct {
	Command []string `json:"command"`
	Timeout int64    `json:"timeout"`
}

// TaskResult is the result of running a task.
type TaskResult struct {
	Command    []string   `json:"command"`
	ExecutedAt int64      `json:"executed_at"`
	DurationMS MSDuration `json:"duration_ms"`
	ExitCode   int        `json:"exit_code"`
	Output     string     `json:"output"`
	Error      string     `json:"error"`
}
