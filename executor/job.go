package executor

import (
	"io"
	"time"

	"github.com/Ensono/taskctl/variables"
)

// Job is a linked list of jobs to execute by Executor
type Job struct {
	Command string
	Dir     string
	Env     *variables.Variables
	Vars    *variables.Variables
	Timeout *time.Duration

	Stdout, Stderr io.Writer
	Stdin          io.Reader

	Next *Job
}

// NewJobFromCommand creates new Job instance from given command
func NewJobFromCommand(command string) *Job {
	return &Job{
		Command: command,
		Vars:    variables.NewVariables(),
		Env:     variables.NewVariables(),
	}
}
