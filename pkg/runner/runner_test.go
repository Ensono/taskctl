package runner

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Ensono/taskctl/pkg/output"
	"github.com/Ensono/taskctl/pkg/utils"
	"github.com/Ensono/taskctl/pkg/variables"

	taskpkg "github.com/Ensono/taskctl/pkg/task"
)

func TestTaskRunner(t *testing.T) {
	c := NewExecutionContext(nil, "/", variables.NewVariables(), &utils.Envfile{}, []string{"true"}, []string{"false"}, []string{"echo 1"}, []string{"echo 2"})

	runner, err := NewTaskRunner(WithContexts(map[string]*ExecutionContext{"local": c}))
	if err != nil {
		t.Fatal(err)
	}
	runner.SetContexts(map[string]*ExecutionContext{
		"default": DefaultContext(),
		"local":   c,
	})
	if _, ok := runner.contexts["default"]; !ok {
		t.Error()
	}

	runner.Stdout, runner.Stderr = io.Discard, io.Discard
	runner.SetVariables(variables.FromMap(map[string]string{"Root": "/tmp"}))
	runner.WithVariable("Root", "/")

	task1 := taskpkg.NewTask("t1")
	task1.Context = "local"
	task1.ExportAs = "EXPORT_NAME"

	task1.Commands = []string{"echo 'taskctl'"}
	task1.Name = "some test task"
	task1.Dir = "{{.Root}}"
	task1.After = []string{"echo 'after task1'"}

	d := 1 * time.Minute
	task2 := taskpkg.NewTask("t2")
	task2.Timeout = &d
	task2.Variations = []map[string]string{{"GOOS": "windows"}, {"GOOS": "linux"}}

	task2.Commands = []string{"false"}
	task2.Name = "some test task"
	task2.Dir = "{{.Root}}"
	task2.Interactive = true

	task3 := taskpkg.NewTask("t3")
	task3.Condition = "exit 1"

	task4 := taskpkg.NewTask("t4")
	task4.Commands = []string{"function test_func() { echo \"BBB\"; } ", "test_func"}

	cases := []struct {
		t                *taskpkg.Task
		skipped, errored bool
		status           int16
		output           string
	}{
		{t: task1, output: "taskctl"},
		{t: task2, status: 1, errored: true},
		{t: task3, status: -1, skipped: true},
		{t: task4, output: "BBB"},
	}

	for _, testCase := range cases {
		err = runner.Run(testCase.t)
		if err != nil && !testCase.errored && !testCase.skipped {
			t.Fatal(err)
		}

		if !testCase.skipped && testCase.t.Start.IsZero() {
			t.Error()
		}

		if !strings.Contains(testCase.t.Output(), testCase.output) {
			t.Error()
		}

		if testCase.errored && !testCase.t.Errored {
			t.Error()
		}

		if !testCase.errored && testCase.t.Errored {
			t.Error()
		}

		if testCase.t.ExitCode != testCase.status {
			t.Error()
		}
	}

	runner.Finish()
}

func Test_DockerExec_Cmd(t *testing.T) {
	ttests := map[string]struct {
		execContext *ExecutionContext
		command     string
	}{
		"runs with default env file": {
			execContext: NewExecutionContext(&utils.Binary{Bin: "docker", Args: []string{
				"run",
				"--rm",
				"alpine", "sh", "-c",
			}}, "/", variables.NewVariables(), utils.NewEnvFile(func(e *utils.Envfile) {
				e.Generate = true
			}), []string{"true"}, []string{"false"}, []string{"echo 1"}, []string{"echo 2"}),
			command: "echo 'taskctl'",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			runner, err := NewTaskRunner(WithContexts(map[string]*ExecutionContext{"default_docker": tt.execContext}))
			if err != nil {
				t.Fatal(err)
			}
			defer runner.Finish()

			testOut, testErr := &bytes.Buffer{}, &bytes.Buffer{}
			runner.Stdout, runner.Stderr = testOut, testErr
			runner.SetVariables(variables.FromMap(map[string]string{"Root": "/tmp"}))
			runner.WithVariable("Root", "/")

			task1 := taskpkg.NewTask("default:docker")
			task1.Context = "default_docker"
			task1.ExportAs = "EXPORT_NAME"

			task1.Commands = []string{tt.command}
			task1.Name = "some test task"
			task1.Dir = "{{.Root}}"
			task1.After = []string{"echo 'after task1'"}

			if err := runner.Run(task1); err != nil {
				fmt.Println(testOut.String())
				t.Fatal(err)
			}

			if len(testErr.String()) > 0 {
				t.Fatalf("got: %s, wanted nil", testErr.String())
			}
		})
	}
}

func ExampleTaskRunner_Run() {
	t := taskpkg.FromCommands("t1", "go doc github.com/Ensono/taskctl/pkg/runner.Runner")
	ob := output.NewSafeWriter(&bytes.Buffer{})
	r, err := NewTaskRunner(func(tr *TaskRunner) {
		tr.Stdout = ob
	})
	if err != nil {
		return
	}
	err = r.Run(t)
	if err != nil {
		fmt.Println(err, t.ExitCode, t.ErrorMessage())
	}
	fmt.Println(ob.String())
	// indentation is important with the matched output here
	// Output: package runner // import "github.com/Ensono/taskctl/pkg/runner"
	//
	// type Runner interface {
	// 	Run(t *task.Task) error
	// 	Cancel()
	//	Finish()
	// }
	//     Runner describes tasks runner interface
}

func TestTaskRunner_ResetContext_WithVariations(t *testing.T) {

	ttests := map[string]struct {
		resetContext bool
		want         string
		variations   []map[string]string
	}{
		"noreset:context": {
			false,
			"first\nfirst\nfirst\nfirst\n",
			[]map[string]string{
				{"Var1": "first"}, {"Var1": "second"},
				{"Var1": "third"}, {"Var1": "fourth"},
			},
		},
		"withreset:context": {
			true,
			"first\nsecond\nthird\nfourth\n",
			[]map[string]string{
				{"Var1": "first"}, {"Var1": "second"},
				{"Var1": "third"}, {"Var1": "fourth"},
			},
		},
	}

	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			task := taskpkg.NewTask(name)
			task.Commands = []string{"echo $Var1"}
			task.ResetContext = tt.resetContext // this is set by defualt but setting here for clarity
			task.Variations = tt.variations

			r, err := NewTaskRunner()
			if err != nil {
				t.Fatal(err)
			}

			ob, eb := &bytes.Buffer{}, &bytes.Buffer{}
			r.Stderr = eb
			r.Stdout = ob

			if err := r.Run(task); err != nil {
				t.Fatal(err)
			}

			if len(task.Output()) < 1 {
				t.Error("nothing written")
			}
			if string(task.Output()) != tt.want {
				t.Errorf("\ngot:\n%s\nwant:\n%s", task.Output(), tt.want)
			}
		})
	}
}
