package cmdutils

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/charmbracelet/huh"
	"github.com/logrusorgru/aurora"
)

func DisplayTaskSelection(conf *config.Config) (taskOrPipelineSelected string, err error) {
	optionMap := []huh.Option[string]{}

	for pipeline := range conf.Pipelines {
		optionMap = append(optionMap, huh.NewOption(fmt.Sprintf("%s - %s", pipeline, aurora.Gray(12, "pipeline")), pipeline))
	}

	for _, task := range conf.Tasks {
		optionMap = append(optionMap, huh.NewOption(fmt.Sprintf("%s - %s", task.Name, aurora.Gray(12, task.Description)), task.Name)) // fmt.Sprintf("Task: %s", task.Name)
	}

	taskOrPipelineName := huh.NewForm(
		huh.NewGroup(
			// select file name
			huh.NewSelect[string]().
				Title("Select the pipelines or tasks to run").
				Options(optionMap...).
				Value(&taskOrPipelineSelected),
		),
	).WithHeight(8).WithShowHelp(true)
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

	fmt.Fprintln(chanOut, aurora.Bold("Summary:").String())

	var log string
	for _, stage := range stages {
		switch stage.ReadStatus() {
		case scheduler.StatusDone:
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Green("- Stage %s was completed in %s"), stage.Name, stage.Duration()))
		case scheduler.StatusSkipped:
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Green("- Stage %s was skipped"), stage.Name))
		case scheduler.StatusError:
			log = strings.TrimSpace(stage.Task.ErrorMessage())
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Red("- Stage %s failed in %s"), stage.Name, stage.Duration()))
			if log != "" {
				fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Red("  > %s"), log))
			}
		case scheduler.StatusCanceled:
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Gray(12, "- Stage %s was cancelled"), stage.Name))
		default:
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Red("- Unexpected status %d for stage %s"), stage.Status, stage.Name))
		}
	}

	fmt.Fprintln(chanOut, aurora.Sprintf("%s: %s", aurora.Bold("Total duration"), aurora.Green(g.Duration())))
}
