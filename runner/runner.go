// package runner
//
// Runner runs the command inside the executor shell
// It uses the mvdan.sh shell implementation in Go.
// injects a custom environment per execution
//
// not all *nix* commands are available, should only be used for a limited number of scenarios
//
// Container Specific implementation runner will use the MobyAPI
package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/output"
	"github.com/Ensono/taskctl/task"
	"github.com/Ensono/taskctl/variables"
	"github.com/sirupsen/logrus"
)

var ErrArtifactFailed = errors.New("artifact not processed")

// Runner describes tasks runner interface
type Runner interface {
	Run(t *task.Task) error
	Cancel()
	Finish()
}

type Executor interface {
	Execute(context.Context, *Job) ([]byte, error)
}

// TaskRunner struct holds the properties and methods
// for running the tasks inside the given executor
type TaskRunner struct {
	Executor  Executor
	DryRun    bool
	contexts  map[string]*ExecutionContext
	variables *variables.Variables
	env       *variables.Variables

	ctx         context.Context
	cancelFunc  context.CancelFunc
	cancelMutex sync.RWMutex
	canceling   bool
	doneCh      chan struct{}

	compiler *TaskCompiler

	Stdin          io.Reader
	Stdout, Stderr io.Writer
	OutputFormat   string

	cleanupList sync.Map
}

// NewTaskRunner creates new TaskRunner instance
func NewTaskRunner(opts ...Opts) (*TaskRunner, error) {
	r := &TaskRunner{
		compiler:     NewTaskCompiler(),
		OutputFormat: string(output.RawOutput),
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		variables:    variables.NewVariables(),
		env:          variables.NewVariables(),
		cancelMutex:  sync.RWMutex{},
		doneCh:       make(chan struct{}, 1),
	}

	r.ctx, r.cancelFunc = context.WithCancel(context.Background())

	for _, o := range opts {
		o(r)
	}

	r.env = variables.FromMap(map[string]string{"ARGS": r.variables.Get("Args").(string)})

	return r, nil
}

// SetContexts sets task runner's contexts
func (r *TaskRunner) SetContexts(contexts map[string]*ExecutionContext) *TaskRunner {
	r.contexts = contexts
	return r
}

// SetVariables sets task runner's variables
func (r *TaskRunner) SetVariables(vars *variables.Variables) *TaskRunner {
	r.variables = vars

	return r
}

// Run runs provided task.
// TaskRunner first compiles task into linked list of Jobs, then passes those jobs to Executor
//
// Env on the runner is global to all tasks
// it is built using the dotenv output only for now
func (r *TaskRunner) Run(t *task.Task) error {

	defer func() {
		r.cancelMutex.RLock()
		if r.canceling {
			close(r.doneCh)
		}
		r.cancelMutex.RUnlock()
	}()

	execContext, err := r.contextForTask(t)
	if err != nil {
		logrus.Debugf("err in execContext: %s\n", err.Error())
		return err
	}

	outputFormat := r.OutputFormat

	var stdin io.Reader
	if t.Interactive {
		outputFormat = string(output.RawOutput)
		stdin = r.Stdin
	}

	taskOutput, err := output.NewTaskOutput(t, outputFormat, r.Stdout, r.Stderr)
	if err != nil {
		return err
	}

	defer func(t *task.Task) {
		err := taskOutput.Finish()
		if err != nil {
			logrus.Error(err)
		}
		taskOutput.Close()

		err = execContext.After()
		if err != nil {
			logrus.Error(err)
		}

		if !t.Errored() && !t.Skipped() {
			t.WithExitCode(0)
		}
	}(t)

	vars := r.variables.Merge(t.Variables)

	env := r.env.Merge(execContext.Env)
	env = env.With("TASK_NAME", t.Name)
	env = env.Merge(t.Env)
	// denormalized graph will append all ancestral env keys to the task
	// if task also includes an envfile property
	// We need to read it in and hang on the env for the command compiler.
	if reader, exists := utils.ReaderFromPath(t.EnvFile); exists {
		m, err := utils.ReadEnvFile(reader)
		if err != nil {
			return fmt.Errorf("%v, %w", err, utils.ErrEnvfileFormatIncorrect)
		}
		// now overwriting any env set properties in the envfile
		env = variables.FromMap(m).Merge(env)
	}

	meets, err := r.checkTaskCondition(t)
	if err != nil {
		return err
	}

	if !meets {
		logrus.Infof("task %s was skipped", t.Name)
		t.WithSkipped(true)
		return nil
	}

	err = r.before(r.ctx, t, env, vars)
	if err != nil {
		return err
	}

	job, err := r.compiler.CompileTask(t, execContext, stdin, taskOutput.Stdout(), taskOutput.Stderr(), env, vars)
	if err != nil {
		return err
	}

	err = taskOutput.Start()
	if err != nil {
		return err
	}

	err = r.execute(r.ctx, t, job)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logrus.Debugf("err is cancelled: %s\n", err.Error())
		}
		return err
	}
	if err := r.storeTaskOutput(t); err != nil {
		return err
	}

	return r.after(r.ctx, t, env, vars)
}

// Cancel cancels execution
func (r *TaskRunner) Cancel() {
	r.cancelMutex.Lock()
	if !r.canceling {
		r.canceling = true
		defer logrus.Debug("runner has been cancelled")
		r.cancelFunc()
	}
	r.cancelMutex.Unlock()
	<-r.doneCh
}

// Finish makes cleanup tasks over contexts
func (r *TaskRunner) Finish() {
	r.cleanupList.Range(func(key, value interface{}) bool {
		value.(*ExecutionContext).Down()
		return true
	})
}

// WithVariable adds variable to task runner's variables list.
// It creates new instance of variables container.
func (r *TaskRunner) WithVariable(key, value string) *TaskRunner {
	r.variables = r.variables.With(key, value)

	return r
}

func (r *TaskRunner) before(ctx context.Context, t *task.Task, env, vars *variables.Variables) error {
	if len(t.Before) == 0 {
		return nil
	}

	execContext, err := r.contextForTask(t)
	if err != nil {
		return err
	}

	for _, command := range t.Before {
		job, err := r.compiler.CompileCommand(t.Name, command, execContext, t.Dir, t.Timeout, nil, r.Stdout, r.Stderr, env, vars)
		if err != nil {
			return fmt.Errorf(`"before\" command compilation failed: %w`, err)
		}

		exec, err := GetExecutorFactory(execContext, job)
		if err != nil {
			return err
		}

		_, err = exec.Execute(ctx, job)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TaskRunner) after(ctx context.Context, t *task.Task, env, vars *variables.Variables) error {
	if len(t.After) == 0 {
		return nil
	}

	execContext, err := r.contextForTask(t)
	if err != nil {
		return err
	}

	for _, command := range t.After {
		job, err := r.compiler.CompileCommand(t.Name, command, execContext, t.Dir, t.Timeout, nil, r.Stdout, r.Stderr, env, vars)
		if err != nil {
			return fmt.Errorf(`"after" command compilation failed: %w`, err)
		}

		exec, err := GetExecutorFactory(execContext, job)
		if err != nil {
			return err
		}

		_, err = exec.Execute(ctx, job)
		if err != nil {
			logrus.Warning(err)
		}
	}

	return nil
}

// contextForTask initializes a default or returns an initialized context from config.
//
// It checks whether there is a `taskctl.env` in the cwd if so it ingests it
// and merges with the specified env.
func (r *TaskRunner) contextForTask(t *task.Task) (*ExecutionContext, error) {

	context := DefaultContext()
	if t.Context != "" {
		var ok bool
		if context, ok = r.contexts[t.Context]; !ok {
			return nil, fmt.Errorf("no such context %s", t.Context)
		}
		r.cleanupList.Store(t.Context, context)
	}

	err := context.Up()
	if err != nil {
		return nil, err
	}

	err = context.Before()
	if err != nil {
		return nil, err
	}

	// This will be run at every task start allowing dynamic changes
	context.Env = context.Env.Merge(utils.DefaultTaskctlEnv())
	return context, nil
}

func (r *TaskRunner) checkTaskCondition(t *task.Task) (bool, error) {
	if t.Condition == "" {
		return true, nil
	}

	executionContext, err := r.contextForTask(t)
	if err != nil {
		return false, err
	}

	job, err := r.compiler.CompileCommand(t.Name, t.Condition, executionContext, t.Dir, t.Timeout, nil, r.Stdout, r.Stderr, r.env, r.variables)
	if err != nil {
		return false, err
	}

	exec, err := GetExecutorFactory(executionContext, job)
	if err != nil {
		return false, err
	}

	_, err = exec.Execute(r.ctx, job)
	if err != nil {
		if _, ok := IsExitStatus(err); ok {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (r *TaskRunner) storeTaskOutput(t *task.Task) error {
	// don't do anything if no artifacts are assigned
	if t.Artifacts == nil {
		return nil
	}
	if t.Artifacts.Type == task.DotEnvArtifactType {
		b, err := os.Open(t.Artifacts.Path)
		if err != nil {
			return fmt.Errorf("failed to open, %v\n%w", err, ErrArtifactFailed)
		}
		dotEnvVars, err := utils.ReadEnvFile(b)
		if err != nil {
			return err
		}
		for envKey, envVar := range dotEnvVars {
			r.env.Set(envKey, envVar)
		}
	}
	return nil
}

// execute
func (r *TaskRunner) execute(ctx context.Context, t *task.Task, job *Job) error {
	execContext, err := r.contextForTask(t)
	if err != nil {
		return err
	}

	exec, err := GetExecutorFactory(execContext, job)
	if err != nil {
		return err
	}
	exec.WithReset(t.ResetContext)
	if err != nil {
		return err
	}

	t.WithStart(time.Now())

	for nextJob := job; nextJob != nil; nextJob = nextJob.Next {
		if _, err := exec.Execute(ctx, nextJob); err != nil {
			logrus.Debug(err.Error())
			if status, ok := IsExitStatus(err); ok {
				t.WithExitCode(int16(status))
				if t.AllowFailure {
					t.WithError(err)
					t.WithEnd(time.Now())
					continue
				}
			}
			t.WithError(err)
			t.WithEnd(time.Now())
			return t.Error()
		}
	}
	t.WithEnd(time.Now())

	return nil
}

// Opts is a task runner configuration function.
type Opts func(*TaskRunner)

// WithContexts adds provided contexts to task runner
func WithContexts(contexts map[string]*ExecutionContext) Opts {
	return func(runner *TaskRunner) {
		runner.contexts = contexts
	}
}

// WithVariables adds provided variables to task runner
func WithVariables(variables *variables.Variables) Opts {
	return func(runner *TaskRunner) {
		runner.variables = variables
		runner.compiler.variables = variables
	}
}

// WithGracefulCtx uses the top most context to create child contexts
// this will ensure the cancellation is propagated properly down.
func WithGracefulCtx(ctx context.Context) Opts {
	return func(tr *TaskRunner) {
		tr.ctx, tr.cancelFunc = context.WithCancel(ctx)
	}
}
