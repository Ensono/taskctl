package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/variables"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
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
	execContext *ExecutionContext
}

type ContainerOpts func(*ContainerExecutor)

// NewContainerExecutor initialises an OCI compliant client
//
// It implicitely creates it from `env` any missing vars required to initialise it,
// will be flagged in the error response.
func NewContainerExecutor(execContext *ExecutionContext, opts ...ContainerOpts) (*ContainerExecutor, error) {
	// NOTE: potentially check env vars are set here
	// also cover it in tests to ensure errors are handled correctly
	// os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	ce := &ContainerExecutor{
		cc:          c,
		execContext: execContext,
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

func (e *ContainerExecutor) WithReset(doReset bool) {}

// Execute executes given job with provided context
// Returns job output
func (e *ContainerExecutor) Execute(ctx context.Context, job *Job) ([]byte, error) {
	defer e.cc.Close()

	containerContext := e.execContext.Container()
	cmd := containerContext.ShellArgs
	cmd = append(cmd, job.Command)
	tty, attachStdin := false, false
	if job.Stdin != nil {
		tty = true
		attachStdin = true
	}
	remoteDir := ""
	if e.execContext.Dir != job.Dir {
		remoteDir = job.Dir
	}

	// everything in the container is relative to the `/eirctl` directory
	wd := path.Join("/eirctl", remoteDir)
	// adding the opiniated PWD into the Container Env as per the wd variable
	cEnv := utils.ConvertEnv(utils.ConvertToMapOfStrings(job.Env.Merge(variables.FromMap(map[string]string{"PWD": wd})).Map()))

	containerConfig := &container.Config{
		Image:       containerContext.Name,
		Entrypoint:  containerContext.Entrypoint,
		Env:         cEnv,
		Cmd:         cmd,
		Volumes:     containerContext.Volumes(),
		Tty:         tty,
		AttachStdin: attachStdin,
		// OpenStdin: ,
		// WorkingDir in a container will always be /eirctl
		// will append any job specified paths to the default working
		WorkingDir: wd,
	}
	logrus.Debugf("entrypoint: %v", containerConfig.Entrypoint)
	logrus.Debugf("command: %v", containerConfig.Cmd)
	if err := e.PullImage(ctx, containerContext.Name, job.Stdout); err != nil {
		return nil, err
	}
	logrus.Debugf("%+v", containerConfig.Volumes)
	resp, err := e.cc.ContainerCreate(ctx, containerConfig, nil, nil, nil, "")
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

	out, err := e.cc.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return nil, fmt.Errorf("%v\n%w", err, ErrContainerLogs)
	}
	// capture stderr separately
	stderr := &bytes.Buffer{}
	if _, err := stdcopy.StdCopy(job.Stdout, stderr, out); err != nil {
		return []byte{}, err
	}

	if len(stderr.Bytes()) > 0 {
		errStr := &bytes.Buffer{}
		if _, err = io.Copy(io.MultiWriter(job.Stderr, errStr), stderr); err != nil {
			return nil, err
		}
		return []byte{}, fmt.Errorf(errStr.String())
	}
	return []byte{}, nil
}

// Container pull images - all contexts that have a container property
func (e *ContainerExecutor) PullImage(ctx context.Context, name string, dstOutput io.Writer) error {
	logrus.Debug(name)
	reader, err := e.cc.ImagePull(ctx, name, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("%v\n%w", err, ErrImagePull)
	}

	defer reader.Close()
	// container.ImagePull is asynchronous.
	// The reader needs to be read completely for the pull operation to complete.
	// If stdout is not required, consider using io.Discard instead of os.Stdout.
	// Debug log pull image output
	b := &bytes.Buffer{}
	_, _ = io.Copy(b, reader)
	logrus.Debug(b.String())
	return nil
}

// container attach stdin - via task or context
