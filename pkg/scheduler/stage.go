package scheduler

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/Ensono/taskctl/pkg/variables"
)

// Stage statuses
const (
	StatusWaiting int32 = iota
	StatusRunning
	StatusSkipped
	StatusDone
	StatusError
	StatusCanceled
)

// Stage is a structure that describes execution stage
// Stage is a synonym for a Node in a the unary tree of the execution graph/tree
type Stage struct {
	Name      string
	Condition string
	Task      *task.Task
	Pipeline  *ExecutionGraph
	// Alias is a pointer to the source pipeline
	// this can be referenced multiple times
	// the denormalization process will dereference these
	Alias        string
	DependsOn    []string
	Dir          string
	AllowFailure bool
	status       *atomic.Int32
	env          *variables.Variables
	variables    *variables.Variables
	start        time.Time
	end          time.Time
	mu           sync.Mutex
	Generator    map[string]any
}

// StageOpts is the Node options
//
// Pass in tasks/pipelines or other properties
// using the options pattern
type StageOpts func(*Stage)

func NewStage(name string, opts ...StageOpts) *Stage {
	s := &Stage{
		// Name:      name,
		variables: variables.NewVariables(),
		env:       variables.NewVariables(),
	}
	// Apply options if any
	for _, o := range opts {
		o(s)
	}
	s.Name = name
	// always overwrite and set Status here
	s.status = &atomic.Int32{}
	return s
}

func (s *Stage) FromStage(originalStage *Stage, existingGraph *ExecutionGraph, ancestralParents []string) {
	s.Condition = originalStage.Condition
	s.Dir = originalStage.Dir
	s.AllowFailure = originalStage.AllowFailure
	s.Generator = originalStage.Generator
	// top level env vars
	if existingGraph != nil {
		s.env = s.env.Merge(variables.FromMap(existingGraph.Env))
	}
	s.env = s.env.Merge(originalStage.env)
	s.variables = s.variables.Merge(originalStage.variables)

	if originalStage.Task != nil {
		tsk := task.NewTask(utils.CascadeName(ancestralParents, originalStage.Task.Name))
		tsk.FromTask(originalStage.Task)
		tsk.Env = tsk.Env.Merge(variables.FromMap(existingGraph.Env))
		s.Task = tsk
	}
	if originalStage.Pipeline != nil {
		// error can be ignored as we have already checked it
		pipeline, _ := NewExecutionGraph(
			utils.CascadeName(ancestralParents, originalStage.Pipeline.Name()),
		)
		pipeline.Env = utils.ConvertToMapOfStrings(variables.FromMap(existingGraph.Env).Merge(variables.FromMap(originalStage.Pipeline.Env)).Map())
		s.Pipeline = pipeline
	}

	s.DependsOn = []string{}

	for _, v := range originalStage.DependsOn {
		s.DependsOn = append(s.DependsOn, utils.CascadeName(ancestralParents, v))
	}
}

func (s *Stage) WithEnv(v *variables.Variables) {
	s.env.MergeV2(v)
}

func (s *Stage) Env() *variables.Variables {
	return s.env
}

func (s *Stage) WithVariables(v *variables.Variables) {
	s.variables.MergeV2(v)
}

func (s *Stage) Variables() *variables.Variables {
	return s.variables
}

func (s *Stage) WithStart(v time.Time) *Stage {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.start = v
	return s
}

func (s *Stage) Start() time.Time {
	return s.start
}

func (s *Stage) WithEnd(v time.Time) *Stage {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.end = v
	return s
}

func (s *Stage) End() time.Time {
	return s.end
}

// UpdateStatus updates stage's status atomically
func (s *Stage) UpdateStatus(status int32) {
	s.status.Store(status)
}

// ReadStatus is a helper to read stage's status atomically
func (s *Stage) ReadStatus() int32 {
	return s.status.Load()
}

// Duration returns stage's execution duration
func (s *Stage) Duration() time.Duration {
	return s.end.Sub(s.start)
}

// Keep reference slice for later
// type StageTimeTaken []*Stage

// func (s StageTimeTaken) Len() int {
// 	return len(s)
// }

// func (s StageTimeTaken) Less(i, j int) bool {
// 	return int(s[j].Duration()) > int(s[i].Duration())
// }

// func (s StageTimeTaken) Swap(i, j int) {
// 	s[i], s[j] = s[j], s[i]
// }
