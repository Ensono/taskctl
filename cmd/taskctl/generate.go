package cmd

import (
	"fmt"
	"os"

	"github.com/Ensono/taskctl/internal/cmdutils"
	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/internal/genci"
	"github.com/Ensono/taskctl/internal/utils"

	"github.com/spf13/cobra"
	// yamlv3 "gopkg.in/yaml.v3"
)

type generateFlags struct {
	targetTyp string
}

func newGenerateCmd(rootCmd *TaskCtlCmd) {
	f := &generateFlags{}
	c := &cobra.Command{
		Use:          "generate",
		Aliases:      []string{"ci", "gen-ci"},
		Short:        `generate <pipeline>`,
		Example:      `taskctl generate pipeline1`,
		Args:         cobra.MinimumNArgs(0),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			// if len(f.targetTyp) == 0 {
			// 	return fmt.Errorf("target ")
			// }
			// display selector if nothing is supplied
			if len(args) == 0 {
				selected, err := cmdutils.DisplayTaskSelection(conf, true)
				if err != nil {
					return err
				}
				args = append([]string{selected}, args[0:]...)
			}

			_, argsStringer, err := rootCmd.buildTaskRunner(args, conf)
			if err != nil {
				return err
			}
			return generateDefinition(conf, argsStringer, genci.CITarget(f.targetTyp))
		},
	}
	c.Flags().StringVarP(&f.targetTyp, "target", "t", "", "Target type of the generation. Valid values include github, etc...")
	_ = c.MarkFlagRequired("target")
	rootCmd.Cmd.AddCommand(c)
}

func generateDefinition(conf *config.Config, argsStringer *argsToStringsMapper, implTyp genci.CITarget) (err error) {
	pipeline := argsStringer.pipelineName
	if pipeline == nil {
		return fmt.Errorf("specified arg is not a pipeline")
	}

	genci, err := genci.New(implTyp, conf, pipeline)
	if err != nil {
		return err
	}
	b, err := genci.Convert(conf, pipeline)
	if err != nil {
		return err
	}

	f, err := os.Create(fmt.Sprintf(".github/workflows/%s.yml", utils.ConvertStringToMachineFriendly(pipeline.Name())))
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	return nil
}
