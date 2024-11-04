package config

import (
	"fmt"

	"github.com/Ensono/taskctl/pkg/variables"

	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
)

func buildPipeline(g *scheduler.ExecutionGraph, stages []*PipelineDefinition, cfg *Config) (*scheduler.ExecutionGraph, error) {
	for _, def := range stages {
		var stageTask *task.Task
		var stagePipeline *scheduler.ExecutionGraph

		if def.Task != "" {
			stageTask = cfg.Tasks[def.Task]
			if stageTask == nil {
				return nil, fmt.Errorf("stage build failed: no such task %s", def.Task)
			}
			stageTask.Generator = def.Generator
		} else {
			stagePipeline = cfg.Pipelines[def.Pipeline]
			if stagePipeline == nil {
				return nil, fmt.Errorf("stage build failed: no such pipeline %s", def.Task)
			}
			stagePipeline.Generator = def.Generator

		}
		stage := scheduler.NewStage(def.Name, func(s *scheduler.Stage) {
			s.Condition = def.Condition
			s.Task = stageTask
			s.Pipeline = stagePipeline
			s.DependsOn = def.DependsOn
			s.Dir = def.Dir
			s.AllowFailure = def.AllowFailure
			s.Generator = def.Generator
		})
		if stagePipeline != nil && def.Name != "" && def.Pipeline != def.Name {
			stagePipeline.WithAlias(def.Pipeline)
			stage.Alias = def.Pipeline
		}
		stage.WithEnv(variables.FromMap(def.Env))
		vars := variables.FromMap(def.Variables)
		vars.Set(".Stage.Name", def.Name)
		stage.WithVariables(vars)

		if stage.Dir != "" {
			stage.Task.Dir = stage.Dir
		}

		if stage.Name == "" {
			if def.Task != "" {
				stage.Name = def.Task
			}

			if def.Pipeline != "" {
				stage.Name = def.Pipeline
			}

			if stage.Name == "" {
				return nil, fmt.Errorf("stage for task %s must have name", def.Task)
			}
		}

		if _, err := g.Node(stage.Name); err == nil {
			return nil, fmt.Errorf("stage with same name %s already exists", stage.Name)
		}

		err := g.AddStage(stage)
		if err != nil {
			return nil, err
		}
	}

	return g, nil
}
