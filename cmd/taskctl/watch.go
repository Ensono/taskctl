package cmd

import (
	"sync"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/internal/watch"
	"github.com/Ensono/taskctl/pkg/runner"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newWatchCmd(parentCmd *cobra.Command, configFunc func() (*config.Config, error), taskRunnerFunc func(args []string, conf *config.Config) (*runner.TaskRunner, *argsToStringsMapper, error)) {
	rc := &cobra.Command{
		Use:   "watch",
		Short: `watch [WATCHERS...]`,
		Long:  "starts watching for filesystem events",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := configFunc()
			if err != nil {
				return err
			}
			taskRunner, _, err := taskRunnerFunc(args, conf)
			if err != nil {
				return err
			}

			var wg sync.WaitGroup
			for _, w := range conf.Watchers {
				wg.Add(1)

				go func(w *watch.Watcher) {
					<-cancel
					w.Close()
				}(w)

				go func(w *watch.Watcher) {
					defer wg.Done()

					err := w.Run(taskRunner)
					if err != nil {
						logrus.Error(err)
					}
				}(w)
			}

			wg.Wait()

			return nil
		},
	}
	parentCmd.AddCommand(rc)
}
