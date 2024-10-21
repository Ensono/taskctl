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

// strategy - selector
type GenCi struct {
}

func New() *GenCi {
	return &GenCi{}
}

// mapper
// writer
