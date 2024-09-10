package cmdutils

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/charmbracelet/huh"
)

const (
	MAGENTA_TERMINAL = "\x1b[35m%s\x1b[0m"
	GREEN_TERMINAL   = "\x1b[32m%s\x1b[0m"
	CYAN_TERMINAL    = "\x1b[36m%s\x1b[0m"
	RED_TERMINAL     = "\x1b[31m%s\x1b[0m"
	GREY_TERMINAL    = "\x1b[18m%s\x1b[0m"
	BOLD_TERMINAL    = "\x1b[1m%s"
)

func DisplayTaskSelection(conf *config.Config) (taskOrPipelineSelected string, err error) {
	optionMap := []huh.Option[string]{}

	for pipeline := range conf.Pipelines {
		optionMap = append(optionMap, huh.NewOption(fmt.Sprintf("%s - %s", pipeline, fmt.Sprintf(GREY_TERMINAL, "pipeline")), pipeline))
	}

	for _, task := range conf.Tasks {
		optionMap = append(optionMap, huh.NewOption(fmt.Sprintf("%s - %s", task.Name, fmt.Sprintf(GREY_TERMINAL, task.Description)), task.Name)) // fmt.Sprintf("Task: %s", task.Name)
	}

	taskOrPipelineName := huh.NewForm(
		huh.NewGroup(
			// select file name
			huh.NewSelect[string]().
				Title("Select the pipelines or tasks to run").
				Options(optionMap...).
				Value(&taskOrPipelineSelected),
		),
	).WithShowHelp(true)
	err = taskOrPipelineName.Run()
	return
}

// printSummary is a TUI helper
func PrintSummary(g *scheduler.ExecutionGraph, chanOut io.Writer) {
	var stages = make([]*scheduler.Stage, 0)
	for _, stage := range g.Nodes() {
		stages = append(stages, stage)
	}

	sort.Slice(stages, func(i, j int) bool {
		return stages[j].Start.Nanosecond() > stages[i].Start.Nanosecond()
	})

	fmt.Fprintf(chanOut, BOLD_TERMINAL, "Summary: \n")

	var log string
	for _, stage := range stages {
		switch stage.ReadStatus() {
		case scheduler.StatusDone:
			fmt.Fprintf(chanOut, GREEN_TERMINAL, fmt.Sprintf("- Stage %s was completed in %s\n", stage.Name, stage.Duration()))
		case scheduler.StatusSkipped:
			fmt.Fprintf(chanOut, GREEN_TERMINAL, fmt.Sprintf("- Stage %s was skipped\n", stage.Name))
		case scheduler.StatusError:
			log = strings.TrimSpace(stage.Task.ErrorMessage())
			fmt.Fprintf(chanOut, RED_TERMINAL, fmt.Sprintf("- Stage %s failed in %s\n", stage.Name, stage.Duration()))
			if log != "" {
				fmt.Fprintf(chanOut, RED_TERMINAL, fmt.Sprintf("  > %s\n", log))
			}
		case scheduler.StatusCanceled:
			fmt.Fprintf(chanOut, GREY_TERMINAL, fmt.Sprintf("- Stage %s was cancelled\n", stage.Name))
		default:
			fmt.Fprintf(chanOut, RED_TERMINAL, fmt.Sprintf("- Unexpected status %d for stage %s\n", stage.Status, stage.Name))
		}
	}

	fmt.Fprintf(chanOut, "%s: %s\n", fmt.Sprintf(BOLD_TERMINAL, "Total duration"), fmt.Sprintf(GREEN_TERMINAL, g.Duration()))
}