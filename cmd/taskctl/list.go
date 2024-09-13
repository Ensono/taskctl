package cmd

import (
	"fmt"
	"html/template"
	"slices"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/utils"
	"github.com/spf13/cobra"
)

var listTmpl = `Contexts:{{range $context := .Contexts}}
- {{ $context }}{{else}} no contexts {{end}}

Pipelines:
{{- range $pipeline := .Pipelines}}
- {{ $pipeline }}{{else}} no pipelines 
{{end}}

Tasks:
{{- range $task := .Tasks}}
- {{ $task }}{{else}} no tasks 
{{end}}

Watchers:
{{- range $watcher := .Watchers}}
- {{ $watcher }}{{else}} no watchers 
{{end}}
`

type listCmd struct {
	configFunc func() (*config.Config, error)
}

func newListCmd(parentCmd *cobra.Command, configFunc func() (*config.Config, error)) {
	cc := &listCmd{configFunc: configFunc}
	listAllCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{},
		Short:   `lists contexts, pipelines, tasks and watchers`,
		RunE:    cc.runAllE(),
	}
	listPipelines := &cobra.Command{
		Use:   "pipelines",
		Short: `lists pipelines`,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := configFunc()
			if err != nil {
				return err
			}
			for _, name := range utils.MapKeys(conf.Pipelines) {
				fmt.Fprintln(ChannelOut, name)
			}
			return nil
		},
	}
	listTasks := &cobra.Command{
		Use:   "tasks",
		Short: `lists tasks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := configFunc()
			if err != nil {
				return err
			}
			for _, name := range utils.MapKeys(conf.Tasks) {
				fmt.Fprintln(ChannelOut, name)
			}
			return nil
		},
	}
	listWatchers := &cobra.Command{
		Use:   "watchers",
		Short: `lists watchers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := configFunc()
			if err != nil {
				return err
			}
			for _, name := range utils.MapKeys(conf.Watchers) {
				fmt.Fprintln(ChannelOut, name)
			}
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return nil // postRunReset()
		},
	}

	listAllCmd.AddCommand(listPipelines)
	listAllCmd.AddCommand(listTasks)
	listAllCmd.AddCommand(listWatchers)
	parentCmd.AddCommand(listAllCmd)
}

func (c *listCmd) runAllE() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		conf, err := c.configFunc()
		if err != nil {
			return err
		}
		t := template.Must(template.New("list").Parse(listTmpl))

		contexts := utils.MapKeys(conf.Contexts)
		pipelines := utils.MapKeys(conf.Pipelines)
		tasks := utils.MapKeys(conf.Tasks)
		watchers := utils.MapKeys(conf.Watchers)

		slices.Sort(contexts)
		slices.Sort(pipelines)
		slices.Sort(tasks)
		slices.Sort(watchers)

		return t.Execute(ChannelOut, struct {
			Contexts, Pipelines, Tasks, Watchers []string
		}{
			Contexts:  contexts,
			Pipelines: pipelines,
			Tasks:     tasks,
			Watchers:  watchers,
		})
	}
}
