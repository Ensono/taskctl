package config

import (
	"path/filepath"

	"github.com/Ensono/taskctl/pkg/utils"
	"github.com/Ensono/taskctl/pkg/variables"

	"github.com/Ensono/taskctl/pkg/task"
)

func buildTask(def *TaskDefinition, lc *loaderContext) (*task.Task, error) {

	t := task.NewTask(def.Name)

	t.Description = def.Description
	t.Condition = def.Condition
	t.Commands = def.Command
	t.Variations = def.Variations
	t.Dir = def.Dir
	t.Timeout = def.Timeout
	t.AllowFailure = def.AllowFailure
	t.After = def.After
	t.Before = def.Before
	t.Artifacts = def.Artifacts
	t.Context = def.Context
	t.Interactive = def.Interactive
	t.ResetContext = def.ResetContext

	t.Env = variables.FromMap(def.Env).Merge(t.Env)
	t.Variables = variables.FromMap(def.Variables).Merge(t.Variables)

	t.Variables.Set("Context.Name", t.Context)
	t.Variables.Set("Task.Name", t.Name)

	if def.Envfile != nil && def.Envfile.Path != "" {
		filename := def.Envfile.Path
		if !filepath.IsAbs(filename) && lc.Dir != "" {
			filename = filepath.Join(lc.Dir, filename)
		}

		envs, err := utils.ReadEnvFile(filename)
		if err != nil {
			return nil, err
		}

		t.Env.Merge(variables.FromMap(envs))
	}

	return t, nil
}
