package cmd

import (
	"fmt"
	"html/template"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/spf13/cobra"
)

var showTmpl = `
  Name: {{ .Name -}}
{{ if .Description }}
  Description: {{ .Description }}
{{- end }}
  Context: {{ .Context }}
  Commands: 
{{- range .Commands }}
    - {{ . -}}
{{ end -}}
{{ if .Dir }}
  Dir: {{ .Dir }}
{{- end }}
{{ if .Timeout }}
  Timeout: {{ .Timeout }}
{{- end}}
  AllowFailure: {{ .AllowFailure }}
`

type showCmd struct {
	configFunc func() (*config.Config, error)
}

func newShowCmd(parentCmd *cobra.Command, configFunc func() (*config.Config, error)) {
	cc := &showCmd{configFunc: configFunc}
	showCmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{},
		Short:   `shows task's details`,
		Args:    cobra.RangeArgs(1, 1),
		RunE:    cc.runE(),
	}
	parentCmd.AddCommand(showCmd)
}

func (c *showCmd) runE() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		conf, err := c.configFunc()
		if err != nil {
			return err
		}

		t := conf.Tasks[args[0]]
		if t != nil {
			tmpl := template.Must(template.New("show").Parse(showTmpl))
			return tmpl.Execute(ChannelOut, t)
		}
		return fmt.Errorf("%s. %w", args[0], ErrIncorrectPipelineTaskArg)
	}
}
