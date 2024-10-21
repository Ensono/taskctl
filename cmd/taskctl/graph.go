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
	isMermaid   bool
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
			return graphCmdRun(p, rootCmd.ChannelOut, f.leftToRight, f.isMermaid)
		},
	}

	graphCmd.PersistentFlags().BoolVarP(&f.leftToRight, "lr", "", false, "orientates outputted graph left-to-right")
	_ = rootCmd.viperConf.BindPFlag("lr", graphCmd.PersistentFlags().Lookup("lr"))
	graphCmd.PersistentFlags().BoolVarP(&f.isMermaid, "is-mermaid", "", false, "output the graph in mermaid flowchart format")
	_ = rootCmd.viperConf.BindPFlag("is-mermaid", graphCmd.PersistentFlags().Lookup("is-mermaid"))

	rootCmd.Cmd.AddCommand(graphCmd)
}

const pipelineStartKey string = "pipeline:start"

func graphCmdRun(p *scheduler.ExecutionGraph, channelOut io.Writer, isLr bool, isMermaid bool) error {
	g := dot.NewGraph(dot.Directed)
	g.Attr("center", "true")
	if isLr {
		g.Attr("rankdir", "LR")
	}
	g.Node(pipelineStartKey)
	draw(g, p, "", false)
	if isMermaid {
		fmt.Fprintln(channelOut, dot.MermaidFlowchart(g, dot.MermaidTopToBottom))
		return nil
	}
	fmt.Fprintln(channelOut, g.String())
	return nil
}

// draw recursively walks the tree and adds nodes with a correct dependency
// between the nodes (parents => children).
//
// Same nodes can be call
func draw(g *dot.Graph, p *scheduler.ExecutionGraph, parent string, startAdded bool) {
	for _, v := range p.BFSNodesFlattened(scheduler.RootNodeName) {
		if v.Pipeline != nil {
			draw(g, v.Pipeline, v.Pipeline.Name(), startAdded)
		}
		dependants := p.From(v.Name)
		if len(dependants) == 0 && parent != "" {
			if parent, found := g.FindNodeById(parent); found {
				g.Edge(parent, g.Node(v.Name))
			}
			continue
		}
		for _, child := range p.From(v.Name) {
			if !startAdded {
				if parent, found := g.FindNodeById(pipelineStartKey); found {
					g.Edge(parent, g.Node(v.Name))
					startAdded = true
				}
			}
			g.Edge(g.Node(v.Name), g.Node(child))
		}
	}
}
