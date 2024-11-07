package genci_test

import (
	"testing"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/internal/genci"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
)

func TestGenCi_GithubImpl(t *testing.T) {
	sp, _ := scheduler.NewExecutionGraph("foo",
		scheduler.NewStage("s1", func(s *scheduler.Stage) {
			s.Pipeline, _ = scheduler.NewExecutionGraph("dev",
				scheduler.NewStage("one", func(s *scheduler.Stage) {
					s.Task = task.NewTask("t2")
				}))
		}),
		scheduler.NewStage("s2", func(s *scheduler.Stage) {
			s.Task = task.NewTask("t3")
			s.DependsOn = []string{"s1"}
		}))

	gc, err := genci.New("github", &config.Config{
		Pipelines: map[string]*scheduler.ExecutionGraph{"foo": sp},
		Generate: &config.Generator{
			TargetOptions: map[string]any{"github": map[string]any{"on": map[string]any{"push": map[string][]string{"branches": {"foo", "bar"}}}}}},
	})
	if err != nil {
		t.Errorf("failed to generate github, %v\n", err)
	}
	b, err := gc.Convert(&scheduler.ExecutionGraph{})
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("no bytes written")
	}
}
