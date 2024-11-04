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
	env          variables.Container
	variables    variables.Container
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

func (s *Stage) FromStage(_stg *Stage, existingGraph *ExecutionGraph, ancestralParents []string) {
	s.Condition = _stg.Condition
	s.Dir = _stg.Dir
	s.AllowFailure = _stg.AllowFailure
	s.Generator = _stg.Generator
	if existingGraph != nil {
		s.env.Merge(variables.FromMap(existingGraph.Env)).Merge(_stg.env)
		s.variables.Merge(_stg.variables)
	}

	if _stg.Task != nil {
		tsk := task.NewTask(utils.CascadeName(ancestralParents, _stg.Task.Name))
		tsk.FromTask(_stg.Task)
		tsk.Env.Merge(variables.FromMap(existingGraph.Env))
		s.Task = tsk
	}
	if _stg.Pipeline != nil {
		// error can be ignored as we have already checked it
		pipeline, _ := NewExecutionGraph(
			utils.CascadeName(ancestralParents, _stg.Pipeline.Name()),
			// _stg.Pipeline.BFSNodesFlattened(RootNodeName)...,
		)
		pipeline.Env = utils.ConvertToMapOfStrings(variables.FromMap(existingGraph.Env).Merge(variables.FromMap(pipeline.Env)).Map())
		s.Pipeline = pipeline
		// // if part of denormalized graphs the parent could have a depends - which would be lost
		// if len(_stg.DependsOn) == 0 {
		// 	for _, v := range ancestralParents {
		// 		// has depends on in upper nodes
		// 		upperNode, _ := existingGraph.Node(v)
		// 		if upperNode != nil && len(upperNode.DependsOn) > 0 {
		// 			_stg.DependsOn = append(_stg.DependsOn, upperNode.DependsOn...)
		// 			// break out of the loop we don't want to add recursively deps
		// 			break
		// 		}
		// 	}
		// }
	}

	s.DependsOn = []string{}

	for _, v := range _stg.DependsOn {
		s.DependsOn = append(s.DependsOn, utils.CascadeName(ancestralParents, v))
	}
}

func (s *Stage) WithEnv(v variables.Container) {
	s.env.Merge(v)
}

func (s *Stage) Env() variables.Container {
	return s.env
}

func (s *Stage) WithVariables(v variables.Container) {
	s.variables.Merge(v)
}

func (s *Stage) Variables() variables.Container {
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
