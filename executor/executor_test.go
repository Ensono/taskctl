package executor_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Ensono/taskctl/executor"
	"github.com/Ensono/taskctl/output"
)

func TestDefaultExecutor_Execute(t *testing.T) {
	t.Parallel()
	b := &bytes.Buffer{}
	output := output.NewSafeWriter(b)
	e, err := executor.NewDefaultExecutor(nil, output, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	job1 := executor.NewJobFromCommand("echo 'success'")
	to := 1 * time.Minute
	job1.Timeout = &to

	if _, err := e.Execute(context.Background(), job1); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), "success") {
		t.Error()
	}

	job1 = executor.NewJobFromCommand("exit 1")

	_, err = e.Execute(context.Background(), job1)
	if err == nil {
		t.Error()
	}

	if _, ok := executor.IsExitStatus(err); !ok {
		t.Error()
	}

	job2 := executor.NewJobFromCommand("echo {{ .Fail }}")
	_, err = e.Execute(context.Background(), job2)
	if err == nil {
		t.Error()
	}

	job3 := executor.NewJobFromCommand("printf '%s\\nLine-2\\n' '=========== Line 1 ==================' ")
	_, err = e.Execute(context.Background(), job3)
	if err != nil {
		t.Error()
	}
}

func Test_ContainerExecutor(t *testing.T) {
	t.Run("check client does not start with DOCKER_HOST removed", func(t *testing.T) {
		os.Unsetenv("DOCKER_HOST")
		os.Unsetenv("DOCKER_CERT_PATH")
		os.Unsetenv("DOCKER_TLS_VERIFY")
		os.Unsetenv("DOCKER_API_VERSION")

		if os.Getenv("DOCKER_HOST") != "" {
			t.Errorf("variable exists")
		}

		_, err := executor.NewContainerExecutor()
		if err == nil {
			t.Error("got nil ,wanted an error on init")
		}
	})
	t.Run("use alpine:latest", func(t *testing.T) {
		ce, err := executor.NewContainerExecutor()
		if err != nil {
			t.Error(err)
		}

		b, err := ce.Execute(context.TODO(), &executor.Job{})
		if err != nil {
			t.Error(err)
		}
		
	})
}
