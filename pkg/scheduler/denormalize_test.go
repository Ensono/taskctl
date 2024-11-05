package scheduler_test

import (
	"fmt"
	"testing"

	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/Ensono/taskctl/pkg/variables"
)

func TestStageFrom_(t *testing.T) {
	oldStage := scheduler.NewStage("old-stage", func(s *scheduler.Stage) {
		s.DependsOn = []string{"task1"}
		s.Task = task.NewTask("task2")
		s.WithEnv(variables.FromMap(map[string]string{"foo": "bar", "original": "oldVal"}))
		s.WithVariables(variables.FromMap(map[string]string{"var1": "bar", "var2": "oldVal"}))
	})

	g, _ := scheduler.NewExecutionGraph("test-merge", oldStage)
	g.Env = map[string]string{"global": "global-stuff"}
	newStage := scheduler.NewStage("new-stage")
	newStage.FromStage(oldStage, g, []string{"test-merge"})

	if len(newStage.Env().Map()) == 0 {
		t.Fatal("not merged env")
	}
	for k, v := range newStage.Env().Map() {
		fmt.Printf("%s = %v\n", k, v)
	}
}
