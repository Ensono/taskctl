> :warning: **This repo and the fork is archived**, the active development and maintainance of this tool has moved to [eirctl](https://github.com/Ensono/eirctl). 

> Huge thanks to the [original author](https://github.com/trntv) :clap:, whose work has inspired the spin off into [eirctl](https://github.com/Ensono/eirctl)

<p align="center">
<img width="400" src="https://raw.githubusercontent.com/Ensono/taskctl/master/docs/logo.png" alt="taskctl - developer's routine tasks automation toolkit" title="taskctl - developer's routine tasks automation toolkit" />
</p>

# taskctl - concurrent task runner, developer's routine tasks automation toolkit

[![Go Reference](https://pkg.go.dev/badge/github.com/Ensono/taskctl.svg)](https://pkg.go.dev/github.com/Ensono/taskctl)
[![Go Report Card](https://goreportcard.com/badge/github.com/Ensono/taskctl)](https://goreportcard.com/report/github.com/Ensono/taskctl)

[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=Ensono_taskctl&metric=bugs)](https://sonarcloud.io/summary/new_code?id=Ensono_taskctl)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=Ensono_taskctl&metric=sqale_index)](https://sonarcloud.io/summary/new_code?id=Ensono_taskctl)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=Ensono_taskctl&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=Ensono_taskctl)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=Ensono_taskctl&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=Ensono_taskctl)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=Ensono_taskctl&metric=coverage)](https://sonarcloud.io/summary/new_code?id=Ensono_taskctl)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=Ensono_taskctl&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=Ensono_taskctl)

Simple modern alternative to GNU Make. *taskctl* is concurrent task runner that allows you to design you routine tasks and development pipelines in nice and neat way in human-readable format (YAML, JSON or TOML). 
Given a pipeline (composed of tasks or other pipelines) it builds a graph that outlines the execution plan. Each task my run concurrently or cascade.
Beside pipelines, each single task can be started manually or triggered by built-in filesystem watcher.

## Features

- human-readable configuration format (YAML, JSON or TOML)
- concurrent tasks execution
- highly customizable execution plan
- cross platform
- import local or remote configurations
- integrated file watcher (live reload)
- customizable execution contexts
- different output types
- embeddable task runner
- interactive prompt
- handy autocomplete
- and many more...

```yaml
tasks:
  lint:
    command:
      - golint $(go list ./... | grep -v /vendor/)
      - go vet $(go list ./... | grep -v /vendor/)
  
  test:
    allow_failure: true
    command: go test ./....
        
  build:
    command: go build -o bin/app ./...
    env: 
      GOOS: linux
      GOARCH: amd64
    before: rm -rf bin/*

pipelines:
  release:
    - task: lint
    - task: test
    - task: build
      depends_on: [lint, test]
```

According to this plan `lint` and `test` will run concurrently, `build` will start only when both `lint` and `test` finished.

[![asciicast](https://asciinema.org/a/326726.svg)](https://asciinema.org/a/326726)

## Contents  

- [Getting started](./docs/installation.md)

- [Configuration](#configuration)
  - [Global configuration](#global-configuration)
  - [Example](#example)
- [Tasks](#tasks)
  - [Pass CLI arguments to task](#pass-cli-arguments-to-task)
  - [Task's variations](#tasks-variations)
  - [Task's variables](#tasks-variables)
  - [Storing task's output](#storing-tasks-output)
  - [Conditional execution](#task-conditional-execution)
- [Pipelines](#pipelines)
- [Filesystem watchers](./docs/watchers.md)
- [Contexts](#contexts)
- [Output formats](#taskctl-output-formats)
- [Embeddable task runner](#embeddable-task-runner)
  - [Runner](#runner)
  - [Scheduler](#scheduler)
- [FAQ](#faq)
  - [How does it differ from go-task/task?](#how-does-it-differ-from-go-tasktask)
- [Autocomplete](#autocomplete)
- [Similar projects](#similar-projects)
- [How to contribute?](#how-to-contribute)
- [License](#license)

## Configuration
*taskctl* uses config file (`tasks.yaml` or `taskctl.yaml`) where your tasks and pipelines stored. 
Config file includes following sections:
- tasks
- pipelines
- watchers
- contexts
- variables

Config file may import other config files, directories or URLs.
```yaml
import:
- .tasks/database.yaml
- .tasks/lint/
- https://raw.githubusercontent.com/taskctl/taskctl/master/docs/example.yaml
```

### Example
Config file [example](https://github.com/Ensono/taskctl/blob/master/docs/example.yaml)

### Global configuration
*taskctl* has global configuration stored in ``$HOME/.taskctl/config.yaml`` file. It is handy to store system-wide tasks, reusable contexts, defaults etc. 

## Tasks
Task is a foundation of *taskctl*. It describes one or more commands to run, their environment, executors and attributes such as working directory, execution timeout, acceptance of failure, etc.
```yaml
tasks:
    lint:
        allow_failure: true
        command:
          - golint $(go list ./... | grep -v /vendor/)
          - go vet $(go list ./... | grep -v /vendor/)
          
    build:
        command: go build ./...
        env: 
          GOOS: linux
          GOARCH: amd64
        env_file: /data/.env
        after: rm -rf tmp/*
        variations:
          - GOARCH: amd64
          - GOARCH: arm
            GOARM: 7
```

Task definition takes following parameters:
- `command` - one or more commands to run
- `variations` - list of variations (env variables) to apply to command
- `context` - execution context's name
- `env` - environment variables. All existing environment variables will be passed automatically
- `envfile` - see [envfile]()
- `dir` - working directory. Current working directory by default
- `timeout` - command execution timeout (default: none)
- `allow_failure` - if set to `true` failed commands will not interrupt execution (default: `false`)
- `after` - command that will be executed after command completes
- `before` - command that will be executed before task starts
- `exportAs` - env variable name to store task's output (default: `TASK_NAME_OUTPUT`, where `TASK_NAME` is actual task's name)
- `condition` - condition to check before running task
- `variables` - task's variables
- `interactive` - if `true` provides STDIN to commands (default: `false`)

### Tasks variables
Each task, stage and context has variables to be used to render task's fields  - `command`, `dir`.
Along with globally predefined, variables can be set in a task's definition.
You can use those variables according to `text/template` [documentation](https://golang.org/pkg/text/template/).

Predefined variables are:
- `.Root` - root config file directory
- `.Dir` - config file directory
- `.TempDir` - system's temporary directory
- `.Args` - provided arguments as a string
- `.ArgsList` - array of provided arguments
- `.Task.Name` - current task's name
- `.Context.Name` - current task's execution context's name
- `.Stage.Name` - current stage's name
- `.Output` - previous command's output
- `.Tasks.Task1.Output` - `task1` last command output

Variables can be used inside task definition. For example:
```yaml
tasks:
    task1:
        dir: "{{ .Root }}/some-dir"
        command:
          - echo "My name is {{ .Task.Name }}"
          - echo {{ .Output }} # My name is task1
          - echo "Sleep for {{ .sleep }} seconds"
          - sleep {{ .sleep | default 10 }}
          - sleep {{ .sleep }}
        variables:
          sleep: 3
```

### Pass CLI arguments to task
Any command line arguments succeeding `--` are passed to each task via `.Args`, `.ArgsList` variables or `ARGS` environment variable.

Given this definition:
```yaml
lint1:
  command: go lint {{.Args}}

lint2:
  command: go lint {{index .ArgsList 1}}
```
the resulting command is:
```
$ taskctl lint1 -- package.go
# go lint package.go

$ taskctl lint2 -- package.go main.go
# go lint main.go
```

### Storing task's output
Task output automatically stored to the variable named like this - ``.Tasks.TaskName.Output``, where `TaskName` is the actual task's name.
It is also stored to `TASK_NAME_OUTPUT` environment variable. It's name can be changed by a task's `exportAs` parameter.
Those variables will be available to all dependent stages.

### Tasks variations
Task may run in one or more variations. Variations allows to reuse task with different env variables:
```yaml
tasks:
  build:
    command:
      - GOOS=${GOOS} GOARCH=amd64 go build -o bin/taskctl_${GOOS} ./cmd/taskctl
    env:
      GOFLAGS: -ldflags=-s -ldflags=-w
    variations:
      - GOOS: linux
      - GOOS: darwin
      - GOOS: windows
```
this config will run build 3 times with different GOOS

### Task conditional execution
The following task will run only when there are any changes that are staged but not committed:
```yaml
tasks:
  build:
    command:
      - ...build...
    condition: git diff --exit-code
```

## Pipelines
Pipeline is a set of stages (tasks or other pipelines) to be executed in a certain order. Stages may be executed in parallel or one-by-one. 
Stage may override task's environment, variables etc. 

This pipeline:
```yaml
pipelines:
    pipeline1:
        - task: start task
        - task: task A
          depends_on: "start task"
        - task: task B
          depends_on: "start task"
        - task: task C
          depends_on: "start task"
        - task: task D
          depends_on: "task C"
        - task: task E
          depends_on: ["task A", "task B", "task D"]
        - task: finish
          depends_on: ["task E"]    
```

will result in an execution plan like this: 

```mermaid
flowchart TD
    A(start task) --> B(task A)
    A --> C(task B)
    A --> D(task C)
    B --> E(task E)
    C --> E(task E)
    D --> F(task D)
    F --> E
    E --> X(Finish pipeline)
```

Stage definition takes following parameters:

- `name` - stage name. If not set - referenced task or pipeline name will be used.
- `task` - task to execute on this stage
- `pipeline` - pipeline to execute on this stage
- `env` - environment variables. All existing environment variables will be passed automatically
- `depends_on` - name of stage on which this stage depends on. This stage will be started only after referenced stage is completed.
- `allow_failure` - if `true` failing stage will not interrupt pipeline execution. ``false`` by default
- `condition` - condition to check before running stage
- `variables` - stage's variables

## Taskctl output formats

Taskctl has several output formats:

- `raw` - prints raw commands output
- `prefixed` - strips ANSI escape sequences where possible, prefixes command output with task's name
- `cockpit` - tasks dashboard

## Contexts

Contexts allow you to set up execution environment, variables, binary which will run your task, up/down commands etc.
```yaml
contexts:
  local:
    executable:
      bin: /bin/zsh
      args:
        - -c
    env:
      VAR_NAME: VAR_VALUE
    variables:
      sleep: 10
    quote: "'" # will quote command with provided symbol: "/bin/zsh -c 'echo 1'"
    before: echo "I'm local context!"
    after: echo "Have a nice day!"
```

Context has hooks which may be triggered once before first context usage or every time before task with this context will run.

```yaml
context:
    docker-compose:
      executable:
        bin: docker-compose
        args: ["exec", "api"]
      up: docker-compose up -d api
      down: docker-compose down api

    local:
      after: rm -rf var/*
```

### Docker context

```yaml
  alpine:
    executable:
      bin: /usr/local/bin/docker
      args:
        - run
        - --rm
        - alpine:latest
    env:
      DOCKER_HOST: "tcp://0.0.0.0:2375"
    before: echo "SOME COMMAND TO RUN BEFORE TASK"
    after: echo "SOME COMMAND TO RUN WHEN TASK FINISHED SUCCESSFULLY"

tasks:
  mysql-task:
    context: alpine
    command: uname -a
```

Being able to pass environment variables to a Docker container is crucial for many build scenarios.

## Embeddable task runner

*taskctl* may be embedded into any go program.
Additional information may be found on taskctl's [pkg.go.dev](https://pkg.go.dev/github.com/Ensono/taskctl?tab=overview) page

### Runner

```go
t := task.FromCommands("go fmt ./...", "go build ./..")
r, err := NewTaskRunner()
if err != nil {
    return
}
err  = r.Run(t)
if err != nil {
    fmt.Println(err, t.ExitCode, t.ErrorMessage())
}
fmt.Println(t.Output())
```

### Scheduler

```go
format := task.FromCommands("go fmt ./...")
build := task.FromCommands("go build ./..")
r, _ := runner.NewTaskRunner()
s := NewScheduler(r)

graph, err := NewExecutionGraph(
    &Stage{Name: "format", Task: format},
    &Stage{Name: "build", Task: build, DependsOn: []string{"format"}},
)
if err != nil {
    return
}

err = s.Schedule(graph)
if err != nil {
    fmt.Println(err)
}
```

## FAQ

### How does it differ from go-task/task?

It's amazing how solving same problems lead to same solutions. *taskctl* and go-task have a lot of concepts in common but also have some differences.

1. Main is pipelines. Pipelines and stages allows more precise workflow design because same tasks may have different dependencies (or no dependencies) in different scenarios.
2. Contexts allow you to set up execution environment and binary which will run your task.

## Autocomplete

### Bash

Add to  ~/.bashrc or ~/.profile

```bash
. <(taskctl completion bash)
```

### ZSH

Add to  ~/.zshrc

```zsh
. <(taskctl completion zsh)
```

### Similar projects

- [GNU Make](https://github.com/mirror/make)
- [go-task/task](https://github.com/go-task/task)
- [mage](https://github.com/magefile/mage)
- [tusk](https://github.com/rliebz/tusk)
- [just](https://github.com/casey/just)
- [cr](https://github.com/cirocosta/cr)
- [realize](https://github.com/oxequa/realize)

## How to contribute?

Feel free to contribute in any way you want. Share ideas, submit issues, create pull requests. 
You can start by improving this [README.md](https://github.com/Ensono/taskctl/blob/master/README.md) or suggesting new [features](https://github.com/Ensono/taskctl/issues)
Thank you!

## License

This project is licensed under the GNU GPLv3 - see the [LICENSE.md](LICENSE.md) file for details

## Authors

- Yevhen Terentiev - [trntv](https://github.com/trntv)

See also the list of [contributors](https://github.com/Ensono/taskctl/contributors) who participated in this project.
