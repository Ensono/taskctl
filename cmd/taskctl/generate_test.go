package cmd_test

import "testing"

func Test_generateCommand(t *testing.T) {

	t.Run("errors with pipeline missing", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/generate.yaml", "generate"},
			errored: true,
		})
	})
}
