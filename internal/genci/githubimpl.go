package genci

import (
	"bytes"
	"fmt"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/internal/schema"
	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
	"gopkg.in/yaml.v2"
)

type githubCiImpl struct {
	// TODO: internal props
	conf     *config.Config
	pipeline *scheduler.ExecutionGraph
}

func newGithubCiImpl(conf *config.Config, pipeline *scheduler.ExecutionGraph) *githubCiImpl {
	return &githubCiImpl{
		conf:     conf,
		pipeline: pipeline,
	}
}

func (impl *githubCiImpl) convert() ([]byte, error) {

	ghaWorkflow := &schema.GithubWorkflow{
		Name: utils.ConvertStringToHumanFriendly(impl.pipeline.Name()),
		Jobs: yaml.MapSlice{},
	}
	if gh, err := extractGeneratorMetadata[schema.GithubWorkflow](impl.conf.Generate.TargetOptions); err == nil {
		if gh.On != nil {
			ghaWorkflow.On = gh.On
		}
		if gh.Env != nil {
			ghaWorkflow.Env = gh.Env
		}
	}

	if err := jobLooper(ghaWorkflow, impl.pipeline); err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	enc := yaml.NewEncoder(b)
	defer enc.Close()
	if err := enc.Encode(ghaWorkflow); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

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
			Env:    utils.ConvertToMapOfStrings(node.Env().Map()),
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
