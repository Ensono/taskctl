package runner_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/output"
	"github.com/Ensono/taskctl/runner"
	"github.com/Ensono/taskctl/variables"
)

func TestDefaultExecutor_Execute(t *testing.T) {
	t.Parallel()
	b1 := &bytes.Buffer{}
	output := output.NewSafeWriter(b1)

	job1 := runner.NewJobFromCommand("echo 'success'")
	to := 1 * time.Minute
	job1.Timeout = &to
	job1.Stdout = output

	e, err := runner.GetExecutorFactory(&runner.ExecutionContext{}, job1)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.Execute(context.Background(), job1); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), "success") {
		t.Error()
	}

	// b2 := &bytes.Buffer{}
	// o2 := output.NewSafeWriter(b2)

	job1 = runner.NewJobFromCommand("exit 1")
	job1.Stdout = io.Discard
	job1.Stderr = io.Discard

	_, err = e.Execute(context.Background(), job1)
	if err == nil {
		t.Error()
	}

	if _, ok := runner.IsExitStatus(err); !ok {
		t.Error()
	}

	job2 := runner.NewJobFromCommand("echo {{ .Fail }}")
	_, err = e.Execute(context.Background(), job2)
	if err == nil {
		t.Error()
	}

	job3 := runner.NewJobFromCommand("printf '%s\\nLine-2\\n' '=========== Line 1 ==================' ")
	_, err = e.Execute(context.Background(), job3)
	if err != nil {
		t.Error()
	}
}

func Test_ContainerExecutor(t *testing.T) {

	t.Run("check client does not start with DOCKER_HOST removed", func(t *testing.T) {

	})

	t.Run("semi-integration with alpine:latest", func(t *testing.T) {

		execContext := runner.NewExecutionContext(&utils.Binary{}, "", variables.NewVariables(), &utils.Envfile{},
			[]string{}, []string{}, []string{}, []string{}, runner.WithContainerOpts(&utils.Container{
				Name:      "alpine:3",
				Shell:     "sh",
				ShellArgs: []string{"-c"},
			}))

		if dh := os.Getenv("DOCKER_HOST"); dh == "" {
			t.Fatal("ensure your DOCKER_HOST is set correctly")
		}

		ce, err := runner.GetExecutorFactory(execContext, nil)
		if err != nil {
			t.Error(err)
		}

		so := &bytes.Buffer{}
		se := &bytes.Buffer{}
		_, err = ce.Execute(context.TODO(), &runner.Job{Command: `env
ls -lat .
pwd`,
			Env:    variables.NewVariables(),
			Vars:   variables.NewVariables(),
			Stdout: output.NewSafeWriter(so),
			Stderr: output.NewSafeWriter(se),
		})

		if err != nil {
			t.Fatal(err)
		}

		if len(se.Bytes()) > 0 {
			t.Errorf("got error %v, expected nil\n\n", se.String())
		}

		if len(so.Bytes()) == 0 {
			t.Errorf("got (%s) no output, expected stdout\n\n", se.String())
		}
	})
}
