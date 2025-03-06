package utils

// Binary is a structure for storing binary file path and arguments that should be passed on binary's invocation
type Binary struct {
	// Bin is the name of the executable to run
	// it must exist on the path
	// If using a default mvdn.sh context then
	// ensure it is on your path as symlink if you are only using aliases.
	Bin  string   `mapstructure:"bin" yaml:"bin" json:"bin"`
	Args []string `mapstructure:"args" yaml:"args,omitempty" json:"args,omitempty"`
}

func (b *Binary) WithBaseArgs(args []string) *Binary {
	b.Args = append(b.Args, args...)
	return b
}

func (b *Binary) WithShellArgs(args []string) *Binary {
	b.Args = append(b.Args, args...)
	return b
}

func (b *Binary) WithContainerArgs(args []string) *Binary {
	b.Args = append(b.Args, args...)
	return b
}

func (b *Binary) GetArgs() []string {
	return b.Args
}

// Container is the specific context for containers
// only available to docker API compliant implementations
//
// e.g. docker and podman
//
// The aim is to remove some of the boilerplate away from the existing more
// generic context and introduce a specific context for tasks run in containers.
type Container struct {
	// Name is the name of the container
	//
	// can be specified in the following formats
	//
	// - <image-name> (Same as using <image-name> with the latest tag)
	//
	// - <image-name>:<tag>
	//
	// - <image-name>@<digest>
	//
	// If the known runtime is podman it should include the registry domain
	// e.g. `docker.io/alpine:latest`
	Name string `mapstructure:"name" yaml:"name" json:"name"`
	// Entrypoint Overwrites the default ENTRYPOINT of the image
	Entrypoint []string `mapstructure:"entrypoint" yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	// EnableDinD mounts the docker sock...
	//
	// >highly discouraged
	EnableDinD bool `mapstructure:"enable_dind" yaml:"enable_dind,omitempty" json:"enable_dind,omitempty"`
	// ContainerArgs are additional args used for the container supplied by the user
	//
	// e.g. dcoker run (TASKCTL_ARGS...) (CONTAINER_ARGS...) image (command)
	// The internals will strip out any unwanted/forbidden args
	//
	// Args like the switch --privileged and the --volume|-v flag with the value of /var/run/docker.sock:/var/run/docker.sock
	// will be removed.
	ContainerArgs []string `mapstructure:"container_args" yaml:"container_args,omitempty" json:"container_args,omitempty"`
	// Shell will be used to run the command in a specific shell on the container
	//
	// Must exist in the container
	Shell string `mapstructure:"shell" yaml:"shell,omitempty" json:"shell,omitempty"`
	// Args are additional args to pass to the shell if provided.
	// Once you provide the ShellArgs, you must also specify the Shell as well, as there is no reliable way to ensure the default `sh` accepts provided shell arguments
	//
	// Default Shell and ShellArgs are `sh -c`
	//
	// e.g. docker run (TASKCTL_ARGS...) (CONTAINER_ARGS...) image (shell) (SHELL_ARGS...) (command)
	//
	// Example: with powershell could be: `-Command -NonInteractive` along with a custom shell of `pwsh` would result in `pwsh -Command -NonInteractive (command)`
	ShellArgs []string `mapstructure:"shell_args" yaml:"shell_args,omitempty" json:"shell_args,omitempty"`
	// volumes will be extracted from ContainerArgs
	volumes map[string]struct{}
}
