package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Ensono/taskctl/runner"
	"github.com/Ensono/taskctl/variables"
	"github.com/sirupsen/logrus"

	"github.com/Ensono/taskctl/internal/utils"
)

var DefaultContainerExcludes = []string{"PATH", "HOME", "TMPDIR"}

var ErrBuildContextIncorrect = errors.New("build context properties are incorrect")

func buildContext(def *ContextDefinition) (*runner.ExecutionContext, error) {
	dir := def.Dir
	if dir == "" {
		dir = utils.MustGetwd()
	}
	if def.Container != nil && def.Container.Name == "" {
		return nil, fmt.Errorf("either container image must be specified, %w", ErrBuildContextIncorrect)
	}

	if def.Executable != nil && def.Executable.Bin == "" {
		return nil, fmt.Errorf("executable binary must be specified, %w", ErrBuildContextIncorrect)
	}

	if def.Envfile == nil {
		def.Envfile = &utils.Envfile{}
	}

	osEnvVars := variables.FromMap(utils.ConvertFromEnv(os.Environ()))
	userEnvVars := variables.FromMap(def.Env)
	// build an env order is _IMPORTANT_
	// we need to overwrite any existing env vars with the user supplied ones
	buildEnvVars := osEnvVars.Merge(userEnvVars)
	envFile, err := newEnvFile(def.Envfile, def.Container != nil)
	if err != nil {
		return nil, err
	}

	c := runner.NewExecutionContext(
		contextExecutable(def),
		dir,
		buildEnvVars,
		envFile,
		def.Up,
		def.Down,
		def.Before,
		def.After,
		runner.WithQuote(def.Quote), func(c *runner.ExecutionContext) {
			c.Variables = variables.FromMap(def.Variables)
		},
		runner.WithContainerOpts(def.Container),
	)
	return c, nil
}

func newEnvFile(defEnvFile *utils.Envfile, isContainerContext bool) (*utils.Envfile, error) {
	if defEnvFile == nil && !isContainerContext {
		return defEnvFile, nil
	}

	envFile := utils.NewEnvFile(func(e *utils.Envfile) {
		// REMOVED Generate - as we will always generate when the context is container
		// We will always inject env file from path if present
		e.Exclude = defEnvFile.Exclude
		// add default excludes from host to container
		if isContainerContext {
			e.Exclude = append(e.Exclude, DefaultContainerExcludes...)
		}
		e.PathValue = defEnvFile.PathValue
		e.Include = defEnvFile.Include
		e.Modify = defEnvFile.Modify
		e.Quote = defEnvFile.Quote
		e.ReplaceChar = defEnvFile.ReplaceChar
	})
	if err := defEnvFile.Validate(); err != nil {
		return nil, err
	}
	return envFile, nil
}

func contextExecutable(def *ContextDefinition) *utils.Binary {
	if def.Container != nil && def.Container.Name != "" {
		// docker run --rm --env-file $ENVFILE --entrypoint $ENTRYPOINT -v ${PWD}:/workspace/.taskctl  $IMAGE
		// args := def.Container.Image.ContainerArgs
		executable := &utils.Binary{
			IsContainer: true,
			// this can be podman or any other OCI compliant deamon/runtime
			Bin:  "docker",
			Args: []string{},
		}
		// BASE ARGS are a special case
		executable.WithBaseArgs([]string{"run", "--rm", "--env-file"})

		// CONTAINER ARGS these are best left to be tightly controlled
		containerArgs := []string{"-v", "${PWD}:/workspace/.taskctl"}
		if def.Container.Entrypoint != "" {
			containerArgs = append(containerArgs, "--entrypoint", def.Container.Entrypoint)
		}
		if def.Container.EnableDinD {
			containerArgs = append(containerArgs, "-v", "/var/run/docker.sock:/var/run/docker.sock")
		}
		if def.Container.ContainerArgs != nil {
			containerArgs = append(containerArgs, checkForbiddenContainerArgs(def.Container.ContainerArgs)...)
		}
		// always append current workspace and image to run
		containerArgs = append(containerArgs, "-w", "/workspace/.taskctl", def.Container.Name)
		executable.WithContainerArgs(containerArgs)
		// default shell and flag is set
		// if shell is overwritten it should also contain the
		shellArgs := []string{"sh", "-c"}
		if def.Container.Shell != "" {
			// SHELL ARGS
			shellArgs = []string{def.Container.Shell}
			if def.Container.ShellArgs != nil {
				shellArgs = append(shellArgs, def.Container.ShellArgs...)
			} else {
				// user should know that this might not work
				logrus.Warnf("your chosen shell: %s does not include any arguments, usually at least -c as the command gets parsed as string", def.Container.Shell)
			}
		}
		executable.WithShellArgs(shellArgs)
		return executable
	}
	return def.Executable
}

// forbiddenContainerArgsPairs contains the list of string segments
// when foind in containerArgs they should be ignored and removed
var forbiddenContainerArgsPairs = [1]string{"docker.sock:"} // is an array so it's allocated to the stack
var forbiddenContainerArgsSwitches = [1]string{"--privileged"}

func checkForbiddenContainerArgs(cargs []string) []string {
	if len(cargs) == 0 {
		return cargs
	}
	sanitisedContainerArgs := []string{}

	verbotenArgIdx := []int{}

	// need to iterate over the list of forbidden values in pairs
	// since we want to search for a partial match. this loop is required
	for _, verbotenPair := range forbiddenContainerArgsPairs {
		slices.ContainsFunc(cargs, func(s string) bool {
			if strings.Contains(s, verbotenPair) {
				idx := slices.Index(cargs, s)
				// when looking for pairs need to append both the flag and flag value
				//
				verbotenArgIdx = append(verbotenArgIdx, idx-1, idx)
			}
			return false
		})
	}

	for _, verbotenSwitch := range forbiddenContainerArgsSwitches {
		if idx := slices.Index(cargs, verbotenSwitch); idx > -1 {
			verbotenArgIdx = append(verbotenArgIdx, idx)
		}
	}
	// iterate over the original ContainerArgs
	// and exclude any that are forbidden
	for idx, arg := range cargs {
		if !slices.Contains(verbotenArgIdx, idx) {
			sanitisedContainerArgs = append(sanitisedContainerArgs, arg)
		}
	}

	return sanitisedContainerArgs
}
