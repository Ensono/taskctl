package scheduler_test

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/Ensono/taskctl/pkg/variables"
)

func TestStageFrom_originalToNew(t *testing.T) {
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

func TestExecutionGraph_Flatten(t *testing.T) {
	t.Parallel()

	g := helperGraph(t, "graph:pipeline1")
	if g == nil {
		t.Fatal("graph not found")
	}
	if len(g.Nodes()) != 8 {
		t.Errorf("top level graph does not have correct number of top level jobs, got %v wanted %v", len(g.Nodes()), 8)
	}

	if len(g.Children(scheduler.RootNodeName)) != 1 {
		t.Errorf("root job incorrect got %v wanted %v", len(g.Children(scheduler.RootNodeName)), 1)
	}

	if len(g.Children("graph:task3")) != 2 {
		t.Errorf("graph:task3 children incorrect got %v wanted %v", len(g.Children("graph:task3")), 2)
	}

	flattenedStages := map[string]*scheduler.Stage{}

	g.Flatten(scheduler.RootNodeName, []string{g.Name()}, flattenedStages)
	if len(flattenedStages) != 13 {
		t.Errorf("stages incorrectly flattened: got %v wanted %v\n", len(flattenedStages), 13)
	}
	gotStages := []string{}
	for k, v := range flattenedStages {
		if k != v.Name {
			t.Errorf("key should be the same as name got key (%s) and name (%s)\n", k, v.Name)
		}
		gotStages = append(gotStages, k)
	}
	// keep the list a bit wet to ensure changes are maintained
	nodeList := []string{"graph:pipeline1->graph:pipeline3->graph:task3", "graph:pipeline1->dev->task-p2:1",
		"graph:pipeline1->prod->task-p2:1", "graph:pipeline1->graph:pipeline3->graph:task2",
		"graph:pipeline1->graph:task2", "graph:pipeline1->graph:task4", "graph:pipeline1->graph:pipeline3",
		"graph:pipeline1->graph:task3", "graph:pipeline1->dev", "graph:pipeline1->dev->task-p2:2",
		"graph:pipeline1->prod", "graph:pipeline1->graph:task1", "graph:pipeline1->prod->task-p2:2"}
	for _, v := range nodeList {
		if !slices.Contains(gotStages, v) {
			t.Errorf("stage (%s) not found in %q\n", v, gotStages)
		}
	}
}

var ymlInputTester = []byte(`
output: prefixed
contexts:
  podman:
    container:
      name: alpine:latest
    env: 
      GLOBAL_VAR: this is it
      TF_VAR_name_company: ${{ env.COMPANY }}
      TF_VAR_name_project: ${{ env.PROJECT }}
      TF_VAR_name_component: ${{ env.COMPONENT }}
      TF_VAR_region: ${{ env.REGION }}
    envfile:
      exclude:
        - HOME

pipelines:
  prod:
    - pipeline: graph:pipeline2
  graph:pipeline1:
    - task: graph:task2
      depends_on: 
        - graph:task1
    - task: graph:task3
      depends_on: [graph:task1]
    - name: dev
      pipeline: graph:pipeline2
      depends_on: [graph:task3]
    - pipeline: prod
      depends_on: [graph:task3]
    - task: graph:task4
      depends_on:
        - graph:task2
    - task: graph:task1
    - pipeline: graph:pipeline3
      depends_on:
        - graph:task4

  graph:pipeline2:
    - task: task-p2:2
    - task: task-p2:1
      depends_on:
        - task-p2:2

  graph:pipeline3:
    - task: graph:task2
    - task: graph:task3

tasks:
  graph:task1:
    command: |
      for i in $(seq 1 5); do
        echo "hello task 1 - iteration $i"
        sleep 0
      done
    context: podman

  graph:task2:
    command: |
      echo "hello task 2"
    context: podman

  graph:task3:
    command: "echo 'hello, task3!'"
    env:
      FOO: bar

  graph:task4:
    command: | 
      echo "hello, task4"
    context: podman
    env:
      FOO: bar

  task-p2:1:
    command:
      - |
        echo "hello, p2 ${FOO}"
    context: podman
    env:
      FOO: task1

  task-p2:2:
    command:
      - |
        for i in $(seq 1 5); do
          echo "hello, p2 ${FOO} - iteration $i"
          sleep 0
        done
    env:
      FOO: task2
`)

func helperGraph(t *testing.T, name string) *scheduler.ExecutionGraph {
	t.Helper()

	tf, err := os.CreateTemp("", "graph-*.yml")
	if err != nil {
		t.Fatal("failed to create a temp file")
	}
	defer os.Remove(tf.Name())
	if _, err := tf.Write(ymlInputTester); err != nil {
		t.Fatal(err)
	}

	cl := config.NewConfigLoader(config.NewConfig())
	cfg, err := cl.Load(tf.Name())
	return cfg.Pipelines[name]
}
