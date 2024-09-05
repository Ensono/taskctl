package runner

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/Ensono/taskctl/pkg/executor"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/Ensono/taskctl/pkg/utils"
	"github.com/Ensono/taskctl/pkg/variables"
)

// TaskCompiler compiles tasks into jobs for executor
type TaskCompiler struct {
	variables variables.Container
}

// NewTaskCompiler create new TaskCompiler instance
func NewTaskCompiler() *TaskCompiler {
	return &TaskCompiler{variables: variables.NewVariables()}
}

// CompileTask compiles task into Job (linked list of commands) executed by Executor
func (tc *TaskCompiler) CompileTask(t *task.Task, executionContext *ExecutionContext, stdin io.Reader, stdout, stderr io.Writer, env, vars variables.Container) (*executor.Job, error) {
	vars = t.Variables.Merge(vars)
	var job, prev *executor.Job

	for k, v := range vars.Map() {
		if reflect.ValueOf(v).Kind() != reflect.String {
			continue
		}

		v, err := utils.RenderString(v.(string), vars.Map())
		if err != nil {
			return nil, err
		}
		vars.Set(k, v)
	}

	for _, variant := range t.GetVariations() {
		for _, command := range t.Commands {
			j, err := tc.CompileCommand(
				t.Name,
				command,
				executionContext,
				t.Dir,
				t.Timeout,
				stdin,
				stdout,
				stderr,
				env.Merge(variables.FromMap(variant)),
				vars,
			)
			if err != nil {
				return nil, err
			}

			if job == nil {
				job = j
			}

			if prev == nil {
				prev = j
			} else {
				prev.Next = j
				prev = prev.Next
			}
		}
	}

	return job, nil
}

// CompileCommand compiles command into Job
func (tc *TaskCompiler) CompileCommand(
	taskName string,
	command string,
	executionCtx *ExecutionContext,
	dir string,
	timeout *time.Duration,
	stdin io.Reader,
	stdout, stderr io.Writer,
	env, vars variables.Container,
) (*executor.Job, error) {
	j := &executor.Job{
		Timeout: timeout,
		Env:     env,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Vars:    tc.variables.Merge(vars),
	}

	// Look at the executable details and check if the command is running `docker` determine if an Envfile is being generated
	// If it has then check to see if the args contains the --env-file flag and if does modify the path to the envfile
	// if it does not then add the --env-file flag to the args array
	if executionCtx.Executable != nil && strings.Contains(strings.ToLower(executionCtx.Executable.Bin), "docker") && executionCtx.Envfile.Generate {

		// define the filename to hold the envfile path
		filename := ""

		// get the timestamp to use to append to the envfile name
		suffix := fmt.Sprintf("%s_%v.env", strings.ToLower(strings.Replace(taskName, ":", "_", -1)), time.Now().UnixNano())

		// does the args contain the --env-file string
		idx := slices.Index(executionCtx.Executable.Args, "--env-file")
		if idx > -1 {
			// add 1 to the index to update the path
			idx += 1
			filename = fmt.Sprintf("%s_%s", executionCtx.Executable.Args[idx], suffix)
			executionCtx.Executable.Args[idx] = filename
		} else {
			// the envfile has not been added to the args, so this needs to be added in
			// as the docker args order is important, these will be prepended to the array
			filename = fmt.Sprintf("generated_%s", suffix)
			executionCtx.Executable.Args = append([]string{executionCtx.Executable.Args[0], "--env-file", filename}, executionCtx.Executable.Args[1:]...)
		}

		// set the path to the envfile
		executionCtx.Envfile.Path = filepath.Join(executionCtx.Envfile.GeneratedDir, filename)

		// generate the envfile
		err := executionCtx.GenerateEnvfile()
		if err != nil {
			return nil, err
		}
	}

	var c []string
	if executionCtx.Executable != nil {
		c = []string{executionCtx.Executable.Bin}
		c = append(c, executionCtx.Executable.Args...)
		c = append(c, fmt.Sprintf("%s%s%s", executionCtx.Quote, command, executionCtx.Quote))
	} else {
		c = []string{command}
	}

	j.Command = strings.Join(c, " ")

	var err error
	if dir != "" {
		j.Dir = dir
	} else if executionCtx.Dir != "" {
		j.Dir = executionCtx.Dir
	}

	j.Dir, err = utils.RenderString(j.Dir, j.Vars.Map())
	if err != nil {
		return nil, err
	}

	return j, nil
}
