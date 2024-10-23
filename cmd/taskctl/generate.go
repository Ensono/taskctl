package cmd

import (
	"bytes"
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
		Jobs: yamlv2.MapSlice{},
	}
	if gh, err := extractGeneratorMetadata[schema.GithubWorkflow](conf.Generate); err == nil {
		if gh.On != nil {
			ghaWorkflow.On = gh.On
		}
		if gh.Env != nil {
			ghaWorkflow.Env = gh.Env
		}
	}

	if err := jobLooper(ghaWorkflow, pipeline); err != nil {
		return err
	}
	b := &bytes.Buffer{}
	enc := yaml.NewEncoder(b)
	defer enc.Close()
	if err := enc.Encode(ghaWorkflow); err != nil {
		return err
	}

	f, err := os.Create(fmt.Sprintf(".github/workflows/%s.yml", utils.ConvertStringToMachineFriendly(pipeline.Name())))
	if err != nil {
		return err
	}
	if _, err := f.Write(b.Bytes()); err != nil {
		return err
	}
	return nil
}

const (
	DefaultPrereqJobId string = "generated-prereq"
)

func addDefaultStepsToJob(job *schema.GithubJob) {
	// toggle if checkout or not
	_ = job.AddStep(&schema.GithubStep{
		Uses: "actions/checkout@v4",
	})
	// name: 'Install taskctl'
	_ = job.AddStep(&schema.GithubStep{
		Name: "Install taskctl",
		ID:   "install-taskctl",
		Run: `rm -rf /tmp/taskctl-linux-amd64-v1.8.0-alpha-aaaabbbb1234
wget https://github.com/Ensono/taskctl/releases/download/v1.8.0-alpha-aaaabbbb1234/taskctl-linux-amd64 -O /tmp/taskctl-linux-amd64-v1.8.0-alpha-aaaabbbb1234
cp /tmp/taskctl-linux-amd64-v1.8.0-alpha-aaaabbbb1234 /usr/local/bin/taskctl
chmod u+x /usr/local/bin/taskctl`,
		Shell: "bash",
	})
}

func extractGeneratorMetadata[T any](generatorMeta map[string]any) (T, error) {
	typ := new(T)
	if gh, found := generatorMeta["github"]; found {
		b, err := yaml.Marshal(gh)
		if err != nil {
			return *typ, err
		}
		if err := yaml.Unmarshal(b, typ); err != nil {
			return *typ, err
		}
	}
	return *typ, nil
}

func convertTaskToStep(task *task.Task) *schema.GithubStep {

	step := &schema.GithubStep{
		Name: utils.ConvertStringToHumanFriendly(task.Name),
		ID:   utils.ConvertStringToMachineFriendly(task.Name),
		Run:  fmt.Sprintf("taskctl run task %s", task.Name),
		Env:  utils.ConvertToMapOfStrings(task.Env.Map()),
	}
	if gh, err := extractGeneratorMetadata[schema.GithubStep](task.Generator); err == nil {
		if gh.If != "" {
			step.If = gh.If
		}
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
			_ = job.AddStep(convertTaskToStep(node.Task))
		}
	}
}

// jobLooper accepts a list of top level jobs
func jobLooper(ciyaml *schema.GithubWorkflow, pipeline *scheduler.ExecutionGraph) error {
	nodes := pipeline.BFSNodesFlattened(scheduler.RootNodeName)
	for _, node := range nodes {
		jobName := utils.ConvertStringToMachineFriendly(node.Name)
		job := &schema.GithubJob{
			Name:   utils.ConvertStringToHumanFriendly(node.Name),
			RunsOn: "ubuntu-24.04",
			Env:    utils.ConvertToMapOfStrings(node.Env.Map()),
		}
		// Add defaults
		addDefaultStepsToJob(job)

		if node.Pipeline != nil {
			flattenTasksInPipeline(job, node.Pipeline)
		}
		if node.Task != nil {
			_ = job.AddStep(convertTaskToStep(node.Task))
		}

		for _, v := range node.DependsOn {
			job.Needs = append(
				job.Needs,
				utils.ConvertStringToMachineFriendly(v),
			)
		}
		if gh, err := extractGeneratorMetadata[schema.GithubJob](node.Generator); err == nil {
			if gh.If != "" {
				job.If = gh.If
			}
			if gh.Environment != "" {
				job.Environment = gh.Environment
			}
			if gh.RunsOn != "" {
				job.RunsOn = gh.RunsOn
			}
		}
		ciyaml.Jobs = append(ciyaml.Jobs, yaml.MapItem{Key: jobName, Value: job})
		// jm[jobName] = *job
	}
	// TODO: enable yamlv3 using yaml.Node :|
	// yn, err := schema.ToYAMLNode(jm)
	// if err != nil {
	// 	return err
	// }
	// ciyaml.Jobs = *yn
	return nil
}
