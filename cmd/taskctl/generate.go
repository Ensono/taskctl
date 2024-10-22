package cmd

import (
	"fmt"
	"os"

	"github.com/Ensono/taskctl/internal/cmdutils"
	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/internal/schema"
	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	yamlv2 "gopkg.in/yaml.v2"
	// yamlv3 "gopkg.in/yaml.v3"
)

func newGenerateCmd(rootCmd *TaskCtlCmd) {
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
			// display selector if nothing is supplied
			if len(args) == 0 {
				selected, err := cmdutils.DisplayTaskSelection(conf)
				if err != nil {
					return err
				}
				args = append([]string{selected}, args[0:]...)
			}

			_, argsStringer, err := rootCmd.buildTaskRunner(args, conf)
			if err != nil {
				return err
			}
			return generateDefinition(conf, argsStringer)
		},
	}

	rootCmd.Cmd.AddCommand(c)
}

func generateDefinition(conf *config.Config, argsStringer *argsToStringsMapper) (err error) {
	pipeline := argsStringer.pipelineName
	if pipeline == nil {
		return fmt.Errorf("specified arg is not a pipeline")
	}

	ghaWorkflow := &schema.GithubWorkflow{
		Name: utils.ConvertStringToHumanFriendly(pipeline.Name()),
		// TODO: add additional metadata here e.g.
		On: &schema.GithubTriggerEvents{
			Push: schema.GithubPushEvent{
				Branches: []string{"master", "main"},
			},
			PullRequest: schema.GithubPullRequestEvent{
				Branches: []string{"master", "main"},
			},
		},
		// init Jobs to add
		Jobs: yamlv2.MapSlice{},
	}

	if err := jobLooper(ghaWorkflow, pipeline); err != nil {
		return err
	}
	// b := &bytes.Buffer{}
	// if err := schema.WriteOut(b, *ghaWorkflow); err != nil {
	// 	return err
	// }
	b, err := yaml.Marshal(ghaWorkflow)
	if err != nil {
		return err
	}
	f, err := os.Create(".github/workflows/generate-test.yml")
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}

	return nil
}

func convertTaskToStep(task *task.Task) *schema.GithubStep {
	step := &schema.GithubStep{
		Name: utils.ConvertStringToHumanFriendly(task.Name),
		ID:   utils.ConvertStringToMachineFriendly(task.Name),
		Run:  "",
		Env:  utils.ConvertToMapOfStrings(task.Env.Map()),
	}
	for _, before := range task.Before {
		step.Run += fmt.Sprintf("%s\n", before)
	}
	for _, cmd := range task.Commands {
		step.Run += fmt.Sprintf("%s\n", cmd)
	}
	for _, after := range task.After {
		step.Run += fmt.Sprintf("%s\n", after)
	}
	return step
}

func flattenTasksInPipeline(job *schema.GithubJob, graph *scheduler.ExecutionGraph) {
	nodes := graph.BFSNodesFlattened(scheduler.RootNodeName)
	for _, node := range nodes {
		if node.Pipeline != nil {
			flattenTasksInPipeline(job, node.Pipeline)
		}
		if node.Task != nil {
			job.AddStep(convertTaskToStep(node.Task))
		}
	}
}

// jobLooper accepts a list of top level jobs
func jobLooper(ciyaml *schema.GithubWorkflow, pipeline *scheduler.ExecutionGraph) error {
	nodes := pipeline.BFSNodesFlattened(scheduler.RootNodeName)
	// jm := make(map[string]schema.GithubJob)
	for _, node := range nodes {
		// jobName := fmt.Sprintf("%v_%s", idx, utils.ConvertStringToMachineFriendly(node.Name))
		jobName := utils.ConvertStringToMachineFriendly(node.Name)
		job := &schema.GithubJob{
			Name:   utils.ConvertStringToHumanFriendly(node.Name),
			RunsOn: "ubuntu-latest",
			// Steps: steps,
			Env: utils.ConvertToMapOfStrings(node.Env.Map()),
		}
		if node.Pipeline != nil {
			flattenTasksInPipeline(job, node.Pipeline)
		}
		if node.Task != nil {
			job.AddStep(convertTaskToStep(node.Task))
		}

		for _, v := range node.DependsOn {
			job.Needs = append(
				job.Needs,
				utils.ConvertStringToMachineFriendly(v),
			)
		}
		ciyaml.Jobs = append(ciyaml.Jobs, yaml.MapItem{Key: jobName, Value: job})
		// jm[jobName] = *job
	}
	// yn, err := schema.ToYAMLNode(jm)
	// if err != nil {
	// 	return err
	// }
	// ciyaml.Jobs = *yn
	return nil
}
