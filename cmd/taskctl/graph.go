package cmd

import (
	"fmt"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/emicklei/dot"
	"github.com/spf13/cobra"
)

type graphFlags struct {
	leftToRight bool
}

type graphCmd struct {
	configFunc func() (*config.Config, error)
	conf       *config.Config
}

func newGraphCmd(parentCmd *TaskCtlCmd, configFunc func() (*config.Config, error)) {
	f := &graphFlags{}
	command := &graphCmd{
		configFunc: configFunc,
	}
	graphCmd := &cobra.Command{
		Use:     "graph",
		Aliases: []string{"g"},
		Short:   `visualizes pipeline execution graph`,
		Long: `Generates a visual representation of pipeline execution plan.
The output is in the DOT format, which can be used by GraphViz to generate charts.`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: command.preRunE(),
		RunE:    command.runE(f),
	}

	graphCmd.PersistentFlags().BoolVarP(&f.leftToRight, "lr", "", false, "orients outputted graph left-to-right")
	_ = parentCmd.viperConf.BindPFlag("lr", graphCmd.PersistentFlags().Lookup("lr"))

	parentCmd.Cmd.AddCommand(graphCmd)
}

func (c *graphCmd) preRunE() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error
		c.conf, err = c.configFunc()
		return err
	}
}

func (c *graphCmd) runE(f *graphFlags) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		pipelineName := args[0]
		return c.graphCmdRun(pipelineName, c.conf)
	}
}

func (c *graphCmd) graphCmdRun(name string, conf *config.Config) error {

	p := conf.Pipelines[name]
	if p == nil {
		return fmt.Errorf("no such pipeline %s", name)
	}

	g := dot.NewGraph(dot.Directed)
	g.Attr("center", "true")
	isLr := conf.Options.GraphOrientationLeftRight
	if isLr {
		g.Attr("rankdir", "LR")
	}

	draw(g, p)

	fmt.Fprintln(ChannelOut, g.String())

	return nil
}

func draw(g *dot.Graph, p *scheduler.ExecutionGraph) {
	for k, v := range p.Nodes() {
		if v.Pipeline != nil {
			cluster := g.Subgraph(k, dot.ClusterOption{})
			draw(cluster, v.Pipeline)
		}

		for _, from := range p.To(k) {
			g.Edge(g.Node(from), g.Node(k))
		}
	}
}
