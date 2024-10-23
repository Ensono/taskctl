// Package genci generates CI yaml definitions based on the
// taskctl pipeline nodes.
//
// This is a translation layer between taskctl concepts of tasks, pipelines and contexts into the world of CI tools yaml syntax.
// See a list of supported tools and overview [here](https://github.com/Ensono/taskctl/blob/master/docs/ci-generator.md).
//
//	Sample output in github
//	```yaml
//
// jobs:
//
//	```
package genci

import (
	"errors"
	"fmt"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/scheduler"
)

var ErrImplementationNotExist = errors.New("implementation does not exist")

type CITarget string

const (
	GitlabCITarget CITarget = "gitlab"
	GitHubCITarget CITarget = "github"
)

// strategy - selector
type GenCi struct {
	implTyp        CITarget
	implementation GenCiIface
	// conf            *config.Config
	// taskctlPipeline *scheduler.ExecutionGraph
}

type GenCiIface interface {
	convert() ([]byte, error)
}

type Opts func(*GenCi)

func New(implTyp CITarget, conf *config.Config, taskctlPipeline *scheduler.ExecutionGraph, opts ...Opts) (*GenCi, error) {
	gci := &GenCi{
		implTyp: implTyp,
	}

	switch implTyp {
	case GitHubCITarget:
		gci.implementation = newGithubCiImpl(conf, taskctlPipeline)
	case GitlabCITarget:
		gci.implementation = &DefualtCiImpl{}
	// TODO: add more here
	default:
		return nil, fmt.Errorf("%s, %w", implTyp, ErrImplementationNotExist)
	}
	return gci, nil
}

func (g *GenCi) Convert(conf *config.Config, taskctlPipeline *scheduler.ExecutionGraph) ([]byte, error) {
	return g.implementation.convert()
}

type DefualtCiImpl struct{}

func (impl *DefualtCiImpl) convert() ([]byte, error) {
	return nil, nil
}
