package utils

import (
	"slices"
	"strings"
)

// Binary is a structure for storing binary file path and arguments that should be passed on binary's invocation
type Binary struct {
	IsContainer bool `jsonschema:"-"`
	// Bin is the name of the executable to run
	// it must exist on the path
	// If using a default mvdn.sh context then
	// ensure it is on your path as symlink if you are only using aliases.
	Bin  string   `mapstructure:"bin" yaml:"bin" json:"bin"`
	Args []string `mapstructure:"args" yaml:"args,omitempty" json:"args,omitempty"`

	// baseArgs will be the default args always managed by taskctl
	// the last item will always be --envfile
	baseArgs      []string
	containerArgs []string
	shellArgs     []string
}

func (b *Binary) WithBaseArgs(args []string) *Binary {
	b.baseArgs = args
	return b
}

func (b *Binary) WithShellArgs(args []string) *Binary {
	b.shellArgs = args
	return b
}

func (b *Binary) WithContainerArgs(args []string) *Binary {
	b.containerArgs = args
	return b
}

func (b *Binary) BaseArgs() []string {
	return b.baseArgs
}

func (b *Binary) ShellArgs() []string {
	return b.shellArgs
}

func (b *Binary) ContainerArgs() []string {
	return b.containerArgs
}

func (b *Binary) buildContainerArgsWithEnvFile(envfilePath string) []string {
	outArgs := append(b.baseArgs, envfilePath)
	outArgs = append(outArgs, b.containerArgs...)
	outArgs = append(outArgs, b.shellArgs...)
	b.Args = outArgs
	return outArgs
}

func (b *Binary) BuildArgsWithEnvFile(envfilePath string) []string {
	if b.IsContainer {
		return b.buildContainerArgsWithEnvFile(envfilePath)
	}

	// sanitize the bin params
	if slices.Contains([]string{"docker", "podman"}, strings.ToLower(b.Bin)) {
		// does the args contain the --env-file string
		// currently we will always either overwrite or just append the `--env-file flag`
		idx := slices.Index(b.Args, "--env-file")
		// the envfile has been added to the args, need to overwrite the value
		if idx > -1 {
			b.Args[idx+1] = envfilePath
			return b.Args
		}

		// the envfile has NOT been added to the args, so this needs to be added in
		// as the docker args order is important, these will be prepended to the array
		b.Args = append([]string{b.Args[0], "--env-file", envfilePath}, b.Args[1:]...)
		return b.Args
	}
	return b.Args
}
