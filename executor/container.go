package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Ensono/taskctl/runner"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	ErrImagePull       = errors.New("failed to pull container image")
	ErrContainerCreate = errors.New("failed to create container")
	ErrContainerStart  = errors.New("failed to start container")
	ErrContainerWait   = errors.New("failed to wait for container")
	ErrContainerLogs   = errors.New("failed to get container logs")
)

// ContainerExecutorIface interface used by this implementation
type ContainerExecutorIface interface {
	Close() error
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error)
	ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error)
}

type ContainerExecutor struct {
	// containerClient
	cc          ContainerExecutorIface
	execContext *runner.ExecutionContext
}

type ContainerOpts func(*ContainerExecutor)

// NewContainerExecutor initialises an OCI compliant client
//
// It implicitely creates it from `env` any missing vars required to initialise it,
// will be flagged in the error response.
func NewContainerExecutor(execContext *runner.ExecutionContext, opts ...ContainerOpts) (*ContainerExecutor, error) {
	// NOTE: potentially check env vars are set here
	// also cover it in tests to ensure errors are handled correctly
	// os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	ce := &ContainerExecutor{
		cc: c,
	}

	for _, opt := range opts {
		opt(ce)
	}

	return ce, nil
}

func WithClient(client ContainerExecutorIface) ContainerOpts {
	return func(ce *ContainerExecutor) {
		ce.cc = client
	}
}

// WithEnv is used to set more specifically the environment vars inside the executor
func (e *ContainerExecutor) WithEnv(env []string) *ContainerExecutor {
	return e
}

func (e *ContainerExecutor) WithReset(doReset bool) *ContainerExecutor {
	return e
}

// Execute executes given job with provided context
// Returns job output
func (e *ContainerExecutor) Execute(ctx context.Context, job *Job) ([]byte, error) {
	defer e.cc.Close()
	// job.Command

	reader, err := e.cc.ImagePull(ctx, "docker.io/library/alpine", image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("%v\n%w", err, ErrImagePull)
	}

	defer reader.Close()
	// cli.ImagePull is asynchronous.
	// The reader needs to be read completely for the pull operation to complete.
	// If stdout is not required, consider using io.Discard instead of os.Stdout.
	io.Copy(os.Stdout, reader)

	resp, err := e.cc.ContainerCreate(ctx, &container.Config{
		Image:      "alpine",
		Entrypoint: []string{"/usr/bin/env"},
		Env:        []string{"FOO=hellloooo"},
		Cmd: []string{"sh", "-c", `env
ls -lat .
pwd`}, //"env", "&&",
		Volumes: map[string]struct{}{
			".:/workspace/.taskctl": {}},
		// Cmd:        []string{"env && ls -lat ."}, //"env", "&&",
		Tty:        false,
		WorkingDir: "/workspace/.taskctl",
	}, nil, nil, nil, "")

	if err != nil {
		return nil, fmt.Errorf("%v\n%w", err, ErrContainerCreate)
	}

	if err := e.cc.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("%v\n%w", err, ErrContainerStart)
	}

	statusCh, errCh := e.cc.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("%v\n%w", err, ErrContainerWait)
		}
	case <-statusCh:
	}

	out, err := e.cc.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		return nil, fmt.Errorf("%v\n%w", err, ErrContainerLogs)
	}

	stdcopy.StdCopy(job.Stdout, job.Stderr, out)

	return []byte{}, nil
}
