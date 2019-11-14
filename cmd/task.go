package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/runner"
	"strings"
)

func init() {
	runCommand.AddCommand(taskRunCommand)
}

var taskRunCommand = &cobra.Command{
	Use:   "task [task]",
	Short: "Schedule task",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// todo: OnlyValidArgs
		var tname = args[0]
		t, ok := tasks[tname]
		if !ok {
			logrus.Fatalf("unknown task %s", tname)
		}

		taskArgs := args[1:]

		tr := runner.NewTaskRunner(contexts, true, quiet)
		err := tr.RunWithEnv(t, config.ConvertEnv(map[string]string{
			"ARGS": strings.Join(taskArgs, " "),
		}))
		if err != nil {
			logrus.Error(err)
		}

		close(done)
	},
}
