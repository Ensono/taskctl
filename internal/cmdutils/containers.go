package cmdutils

import (
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func RunDockerContainer() {
	ctx := context.Background()
	os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", image.PullOptions{})
	if err != nil {
		panic(err)
	}

	defer reader.Close()
	// cli.ImagePull is asynchronous.
	// The reader needs to be read completely for the pull operation to complete.
	// If stdout is not required, consider using io.Discard instead of os.Stdout.
	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      "alpine",
		Env:        []string{},
		Entrypoint: nil,
		Volumes: map[string]struct{}{
			"./:/workspace/.taskctl": {}},
		Cmd:        []string{"ls", "-lat", "."}, //"env", "&&",
		Tty:        false,
		WorkingDir: "/workspace/.taskctl",
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}

// func ptr[T any](t T) *T {
// 	return &t
// }

// func RunPodmanContainer() {
// 	fmt.Println("Welcome to the Podman Go bindings tutorial")

// 	// Get Podman socket location
// 	sock_dir := os.Getenv("XDG_RUNTIME_DIR")
// 	if sock_dir == "" {
// 		sock_dir = "/var/run"
// 	}

// 	socket := os.Getenv("DOCKER_HOST")

// 	// Connect to Podman socket
// 	ctx, err := bindings.NewConnection(context.Background(), socket)
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// 	// Pull Busybox image (Sample 1)
// 	fmt.Println("Pulling Busybox image...")
// 	_, err = images.Pull(ctx, "docker.io/busybox", &images.PullOptions{})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// 	// Pull Fedora image (Sample 2)
// 	rawImage := "registry.fedoraproject.org/fedora:latest"
// 	fmt.Println("Pulling Fedora image...")
// 	_, err = images.Pull(ctx, rawImage, &images.PullOptions{})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// 	// List images
// 	imageSummary, err := images.List(ctx, &images.ListOptions{})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	var names []string
// 	for _, i := range imageSummary {
// 		names = append(names, i.RepoTags...)
// 	}
// 	fmt.Println("Listing images...")
// 	fmt.Println(names)

// 	// Container create
// 	s := specgen.NewSpecGenerator(rawImage, false)
// 	s.Terminal = ptr(true)
// 	r, err := containers.CreateWithSpec(ctx, s, &containers.CreateOptions{})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// 	// Container start
// 	fmt.Println("Starting Fedora container...")
// 	err = containers.Start(ctx, r.ID, &containers.StartOptions{})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// 	_, err = containers.Wait(ctx, r.ID, &containers.WaitOptions{
// 		Condition: []define.ContainerStatus{define.ContainerStateRunning},
// 	})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// 	// Container list
// 	var latestContainers = 1
// 	containerLatestList, err := containers.List(ctx, &containers.ListOptions{
// 		Last: &latestContainers,
// 	})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	fmt.Printf("Latest container is %s\n", containerLatestList[0].Names[0])

// 	// Container inspect
// 	ctrData, err := containers.Inspect(ctx, r.ID, &containers.InspectOptions{})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	fmt.Printf("Container uses image %s\n", ctrData.ImageName)
// 	fmt.Printf("Container running status is %s\n", ctrData.State.Status)

// 	// Container stop
// 	fmt.Println("Stopping the container...")
// 	err = containers.Stop(ctx, r.ID, &containers.StopOptions{})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	ctrData, err = containers.Inspect(ctx, r.ID, &containers.InspectOptions{})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	fmt.Printf("Container running status is now %s\n", ctrData.State.Status)
// 	return

// }
