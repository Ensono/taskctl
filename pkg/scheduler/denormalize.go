package scheduler

import (
	"strings"

	"github.com/Ensono/taskctl/internal/utils"
)

// StageTable is a simple hash table of denormalized stages into a flat hash table (map)
//
// NOTE: used for read only at this point
type StageTable map[string]*Stage

// DenormalizePipeline performs a DFS traversal on the ExecutionGraph from the root node
//
// In order to be able to call the same pipeline from another pipeline, we want to create a new
// pointer to it, this will avoid race conditions in times/outputs/env vars/etc...
// We can also set separate vars and environment variables
func (g *ExecutionGraph) DenormalizePipeline() (*ExecutionGraph, error) {
	denormalizedGraph, _ := NewExecutionGraph(g.Name())
	flattenedStages := map[string]*Stage{}

	g.flatten(RootNodeName, []string{g.Name()}, flattenedStages)
	// rebuild graph from flatten denormalized stages
	denormalizedGraph.rebuildFromDenormalized(StageTable(flattenedStages))
	return denormalizedGraph, nil
}

func (g *ExecutionGraph) rebuildFromDenormalized(st StageTable) error {
	for _, stage := range st.NthLevelChildren(g.Name(), 1) {
		if stage.Pipeline != nil {
			c := st.NthLevelChildren(stage.Name, 1)
			// There is no chance that at this point there would be a cycle
			// but keep this check here just in case
			ng, err := NewExecutionGraph(stage.Name, c...)
			if err != nil {
				return err
			}
			ng.rebuildFromDenormalized(st)
			stage.Pipeline = ng
		}
		g.AddStage(stage)
	}
	return nil
}

// NthLevelChildren retrieves the nodes by prefix and depth specified
//
// removing the base prefix and looking at the depth of the keyprefix per stage
func (st StageTable) NthLevelChildren(prefix string, depth int) []*Stage {
	prefixParts := strings.Split(prefix, utils.PipelineDirectionChar)
	stages := []*Stage{}
	for key, stageVal := range st {
		if strings.HasPrefix(key, prefix) && key != prefix {
			keyParts := strings.Split(key, utils.PipelineDirectionChar)
			if len(keyParts[len(prefixParts):]) == depth {
				stages = append(stages, stageVal)
			}
		}
	}
	return stages
}

// flatten is a recursive helper function to clone nodes with unique paths
// each new instance will have a separate memory address allocation
func (graph *ExecutionGraph) flatten(nodeName string, ancestralParentNames []string, flattenedStage map[string]*Stage) {
	uniqueName := utils.CascadeName(ancestralParentNames, nodeName)
	if nodeName != RootNodeName {
		originalNode, _ := graph.Node(nodeName)
		clonedStage := NewStage(uniqueName)
		// Task or stage needs adding
		// Dereference the new stage from the original node
		clonedStage.FromStage(originalNode, graph, ancestralParentNames)
		flattenedStage[uniqueName] = clonedStage

		// If the node has a subgraph, recursively clone it with a new prefix
		if originalNode.Pipeline != nil {
			// creating a graph without stages - cannot error here
			subGraphClone, _ := NewExecutionGraph(uniqueName)
			// peek if children are a single pipeline
			peek := originalNode.Pipeline.Children(RootNodeName)
			// its name is likely reused elsewhere
			if len(peek) == 1 {
				for _, peekStage := range peek {
					if peekStage.Pipeline != nil {
						// aliased stage only contains a single item and
						// that is a pipeline we advance  move forward
						peekStage.DependsOn = clonedStage.DependsOn
						peekStage.Name = originalNode.Name
						originalNode = peekStage
					}
				}
			}
			// use alias or name
			for subNode := range originalNode.Pipeline.Nodes() {
				originalNode.Pipeline.flatten(subNode, append(ancestralParentNames, originalNode.Name), flattenedStage)
			}
			clonedStage.Pipeline = subGraphClone
		}
	}

	// Clone each child node, creating unique names based on the current path
	for _, child := range graph.Children(nodeName) {
		graph.flatten(child.Name, ancestralParentNames, flattenedStage)
	}
}
