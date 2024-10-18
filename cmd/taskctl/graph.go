package cmd

import (
	"fmt"
	"io"

	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/emicklei/dot"
	"github.com/spf13/cobra"
)

type graphFlags struct {
	leftToRight bool
}

func newGraphCmd(rootCmd *TaskCtlCmd) {
	f := &graphFlags{}
	graphCmd := &cobra.Command{
		Use:     "graph",
		Aliases: []string{"g"},
		Short:   `visualizes pipeline execution graph`,
		Long: `Generates a visual representation of pipeline execution plan.
The output is in the DOT format, which can be used by GraphViz to generate charts.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			p := conf.Pipelines[args[0]]
			if p == nil {
				return fmt.Errorf("no such pipeline %s", args[0])
			}
			return graphCmdRun(p, rootCmd.ChannelOut, f.leftToRight)
		},
	}

	graphCmd.PersistentFlags().BoolVarP(&f.leftToRight, "lr", "", false, "orients outputted graph left-to-right")
	_ = rootCmd.viperConf.BindPFlag("lr", graphCmd.PersistentFlags().Lookup("lr"))

	rootCmd.Cmd.AddCommand(graphCmd)
}

func graphCmdRun(p *scheduler.ExecutionGraph, channelOut io.Writer, isLr bool) error {
	g := dot.NewGraph(dot.Directed)
	g.Attr("center", "true")
	if isLr {
		g.Attr("rankdir", "LR")
	}
	draw(g, p)
	fmt.Fprintln(channelOut, g.String())
	return nil
}

func draw(g *dot.Graph, p *scheduler.ExecutionGraph) {
	for _, v := range p.Nodes() { //p.BFSNodes(scheduler.RootNodeName) {
		if v.Name == scheduler.RootNodeName {
			
		}
		if v.Pipeline != nil {
			cluster := g.Subgraph(v.Name, dot.ClusterOption{})
			draw(cluster, v.Pipeline)
		}
		for _, from := range p.To(v.Name) {
			g.Edge(g.Node(from), g.Node(v.Name))
		}
	}
}
