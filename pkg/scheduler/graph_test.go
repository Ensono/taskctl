package scheduler_test

import (
	"errors"
	"testing"

	"github.com/Ensono/taskctl/pkg/scheduler"
)

func TestExecutionGraph_AddStage(t *testing.T) {
	t.Parallel()
	g, err := scheduler.NewExecutionGraph("test")
	if err != nil {
		t.Fatal(err)
	}

	err = g.AddStage(scheduler.NewStage("stage1", func(s *scheduler.Stage) {
		s.DependsOn = []string{"stage2"}
	}))
	if err != nil {
		t.Fatal()
	}
	err = g.AddStage(scheduler.NewStage("stage2", func(s *scheduler.Stage) {
		s.DependsOn = []string{"stage1"}
	}))
	if err == nil {
		t.Fatal("add stage cycle detection failed\n")
	}
	if err != nil && !errors.Is(err, scheduler.ErrCycleDetected) {
		t.Fatalf("incorrect error (%q), wanted: %q\n", err, scheduler.ErrCycleDetected)
	}
}
