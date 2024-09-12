package cmd_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	taskctlCmd "github.com/Ensono/taskctl/cmd/taskctl"
	"github.com/Ensono/taskctl/pkg/output"
)

type runTestIn struct {
	args        []string
	errored     bool
	exactOutput string
	output      []string
}

func runTestHelper(t *testing.T, tt *runTestIn) {
	t.Helper()
	taskctlCmd.ChannelOut = nil
	taskctlCmd.ChannelErr = nil
	cmd := taskctlCmd.TaskCtlCmd

	errOut := output.NewSafeWriter(&bytes.Buffer{})
	stdOut := output.NewSafeWriter(&bytes.Buffer{})
	logOut := output.NewSafeWriter(&bytes.Buffer{})
	logErr := output.NewSafeWriter(&bytes.Buffer{})

	// silence output
	taskctlCmd.ChannelOut = logOut
	taskctlCmd.ChannelErr = logErr
	cmdArgs := tt.args

	cmd.SetArgs(cmdArgs)
	cmd.SetErr(errOut)
	cmd.SetOut(stdOut)

	defer func() {
		cmd = nil
		taskctlCmd.ChannelErr = nil
		taskctlCmd.ChannelOut = nil
	}()

	if err := cmd.ExecuteContext(context.TODO()); err != nil {
		if tt.errored {
			return
		}
		t.Errorf("got: %v, wanted <nil>", err)
	}

	if tt.errored && errOut.Len() < 1 {
		t.Errorf("got: nil, wanted an error to be thrown")
	}
	if len(tt.output) > 0 {
		for _, v := range tt.output {
			if !strings.Contains(logOut.String(), v) {
				t.Errorf("\"%s\" not found in \"%s\"", v, logOut.String())
			}
		}
	}
	if tt.exactOutput != "" && logOut.String() != tt.exactOutput {
		t.Errorf("output mismatch, expected = %s, got = %s", tt.exactOutput, logOut.String())
	}
}
