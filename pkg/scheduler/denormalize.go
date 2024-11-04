package scheduler

import (
	"fmt"
	"strings"

	"github.com/Ensono/taskctl/internal/utils"
)

// DenormalizePipelineRefs performs a DFS traversal on the ExecutionGraph from the root node
//
// In order to be able to call the same pipeline from another pipeline, we want to create a new
// pointer to it, this will avoid race conditions in times/outputs/env vars/etc...
// We can also set separate environment variables
func (g *ExecutionGraph) DenormalizedPipeline() (*ExecutionGraph, error) {
	ng := CloneGraph(g)

	// // rootNode, _ := g.Node(RootNodeName)
	// // printNode(g, g.children[RootNodeName], 0)

	// // topLevelNodes := g.BFSNodesFlattened(RootNodeName)
	// // fmt.Println(topLevelNodes)
	// // recreate stages
	// denormalizedStages := map[string]*Stage{}
	// ng, _ := NewExecutionGraph(g.Name())
	// g.dfs([]string{g.Name()}, denormalizedStages, ng)

	// stages := []*Stage{}
	// for _, v := range denormalizedStages {
	// 	// remove empty nodes - i.e. ones without Task or Pipeline(subgrpah)
	// 	// this would happen when denormalizing pipelines of pipelines with different name
	// 	if v.Name == RootNodeName {
	// 		continue
	// 	}
	// 	// if v.Pipeline != nil {
	// 	// 	if len(v.Pipeline.Children(RootNodeName)) == 0 {
	// 	// 		continue
	// 	// 	}
	// 	// }
	// 	dn := strings.Split(v.Name, utils.PipelineDirectionChar)
	// 	if len(dn) > 2 && len(v.DependsOn) == 0 {
	// 		// assign ancestor
	// 		v.DependsOn = append(v.DependsOn, strings.Join(dn[0:len(dn)-1], utils.PipelineDirectionChar))
	// 	}
	// 	if v.Pipeline != nil {
	// 		fmt.Printf("pipeline: %q, stage (%q)\n", v.Pipeline.Name(), v.Name)
	// 	}
	// 	if v.Task != nil {
	// 		fmt.Printf("task: %q, stage (%q), \n", v.Task.Name, v.Name)
	// 	}
	// 	if v.Task == nil && v.Pipeline == nil {
	// 		fmt.Printf("stage name: %q, task and pipeline nil\n", v.Name)
	// 		// continue
	// 	}
	// 	stages = append(stages, v)

	// 	// if v.Task != nil {
	// 	// 	stages = append(stages, v)
	// 	// }
	// 	// if v.Pipeline != nil && len(v.Pipeline.Children(RootNodeName)) > 0 {
	// 	// 	stages = append(stages, v)
	// 	// }

	// }
	// // perform second pass
	// return NewExecutionGraph(g.Name(), stages...)
	return ng, nil
}

// printNode is a recursive helper to print nodes with depth indentation
func printNode(g *ExecutionGraph, children []string, depth int) {
	// Print the current node's prefix with indentation
	indent := strings.Repeat("  ", depth)
	leafMarker := ""
	// if stage != nil && stage.Pipeline != nil {
	// 	leafMarker = "(pipeline)"
	// 	rootStage, _ := stage.Pipeline.Node(RootNodeName)
	// 	fmt.Printf("%s%s%s\n", indent, rootStage.Name, leafMarker)
	// 	printNode(stage.Pipeline, rootStage, depth+2)
	// }
	fmt.Printf("%s%s%s\n", indent, g.name, leafMarker)
	// Recursively print each child node
	// children := g.children[RootNodeName]
	for _, nodeChild := range children {
		stage, _ := g.Node(nodeChild)
		if stage.Pipeline != nil {
			leafMarker = " (pipeline)"
			// rootStage, _ := child.Pipeline.Node(RootNodeName)
			fmt.Printf("%s%s%s\n", indent, stage.Pipeline.Name(), leafMarker)
			printNode(g, g.children[nodeChild], depth+2)
		}
		leafMarker = " (task)"
		fmt.Printf("%s%s%s\n", indent, stage.Name, leafMarker)
		if len(g.children[nodeChild]) > 0 {
			printNode(g, g.children[nodeChild], depth+2)
		}
	}
}

func (g *ExecutionGraph) recurseCopyInto(ancestralParentNames []string, stages map[string]*Stage, ng *ExecutionGraph) {
	for _, currentStageNode := range g.Nodes() {
		if currentStageNode.Pipeline != nil {
			nestedAncestors := append(ancestralParentNames, currentStageNode.Name)
			if currentStageNode.Name != currentStageNode.Pipeline.Name() {
				nestedAncestors = append(ancestralParentNames, currentStageNode.Name, currentStageNode.Pipeline.Name())
			}
			currentStageNode.Pipeline.recurseCopyInto(nestedAncestors, stages, ng)
			ng.AddStage(currentStageNode)
		}
	}
}

func (g *ExecutionGraph) dfs(ancestralParentNames []string, stages map[string]*Stage, ng *ExecutionGraph) {

	// Check if the start node exists in the graph
	if _, exists := g.nodes[RootNodeName]; !exists {
		return
	}

	// Initialize a stack and a visited map
	stack := []string{RootNodeName}
	visited := make(map[string]bool)

	// Perform DFS using the stack
	for len(stack) > 0 {
		// Pop the last node from the stack
		currentNode := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Skip if already visited
		if visited[currentNode] {
			continue
		}

		// Mark the current node as visited
		visited[currentNode] = true

		// Retrieve the current currentStageNode
		currentStageNode := g.nodes[currentNode]

		// Process the current node
		if currentNode != RootNodeName {

			uniqueName := utils.CascadeName(ancestralParentNames, currentNode)
			stg := NewStage(uniqueName)
			// Task or stage needs adding
			stg.FromStage(currentStageNode, g, ancestralParentNames)
			stages[uniqueName] = stg
			// ng.AddStage(stg)

			// If the stage has a subgraph, recursively perform DFS on it
			if currentStageNode.Pipeline != nil {
				fmt.Printf("started pipeline %s\n", currentStageNode.Pipeline.Name())
				nestedAncestors := append(ancestralParentNames, currentStageNode.Name)
				if currentStageNode.Name != currentStageNode.Pipeline.Name() {
					nestedAncestors = append(ancestralParentNames, currentStageNode.Name, currentStageNode.Pipeline.Name())
				}
				currentStageNode.Pipeline.dfs(nestedAncestors, stages, ng)
				fmt.Printf("finished pipeline %s\n", currentStageNode.Pipeline.Name())
			}
		}

		// Push all children of the current node onto the stack
		for _, child := range g.Children(currentNode) {
			if !visited[child.Name] {

				stack = append(stack, child.Name)
			}
		}
	}
}

func (g *ExecutionGraph) build(st StageTable) {
	for _, stage := range st.FirstLevelChildren(g.Name(), 1) {
		if stage.Pipeline != nil {
			// check if children in map
			c := st.FirstLevelChildren(stage.Name, 1)
			ng, _ := NewExecutionGraph(stage.Name, c...)
			ng.build(st)
			stage.Pipeline = ng
			// ng.AddStage(stage)
			// clonedGraph.AddStage(stage)

		}
		g.AddStage(stage)
	}
}

// StageTable is a simple hash table
// used for read only at this point
type StageTable map[string]*Stage

// FirstLevelChildren retrieves the nodes
// by prefix and depth
func (st StageTable) FirstLevelChildren(prefix string, depth int) []*Stage {
	prefixParts := strings.Split(prefix, utils.PipelineDirectionChar)
	stages := []*Stage{}
	for key, stageVal := range st {
		// all nodes have been flatten to a
		if strings.HasPrefix(key, prefix) && key != prefix {
			// prefixedRemoved := strings.Replace(key, prefix+utils.PipelineDirectionChar, "", 1)
			keyParts := strings.Split(key, utils.PipelineDirectionChar)
			if len(keyParts[len(prefixParts):]) == depth {
				stages = append(stages, stageVal)
			}
		}
	}
	return stages
}

// CloneGraph denormalizes a graph by duplicating nodes to maintain unique paths
func CloneGraph(graph *ExecutionGraph) *ExecutionGraph {

	clonedGraph, _ := NewExecutionGraph(graph.Name())

	flattenedStages := make(map[string]*Stage)
	cloneHelper(graph, RootNodeName, []string{graph.Name()}, flattenedStages)
	st := StageTable(flattenedStages)
	clonedGraph.build(st)

	return clonedGraph
}

// cloneHelper is a recursive helper function to clone nodes with unique paths
func cloneHelper(graph *ExecutionGraph, nodeName string, ancestralParentNames []string, flattenedStage map[string]*Stage) string {
	uniqueName := utils.CascadeName(ancestralParentNames, nodeName)

	// If this node is already cloned in this path, return the unique name
	// if stage, _ := clonedGraph.Node(uniqueName); stage != nil {
	// 	return uniqueName
	// }
	if nodeName != RootNodeName {
		// Clone the node
		originalNode, _ := graph.Node(nodeName)
		clonedStage := NewStage(uniqueName)
		// Task or stage needs adding
		clonedStage.FromStage(originalNode, graph, ancestralParentNames)
		flattenedStage[uniqueName] = clonedStage
		// newGraph.AddStage(clonedStage)

		// If the node has a subgraph, recursively clone it with a new prefix
		if originalNode.Pipeline != nil {
			// var subGraphClone *ExecutionGraph
			subGraphClone, _ := NewExecutionGraph(uniqueName)
			// peekPipelineChildren := originalNode.Pipeline.children[RootNodeName]
			// peek children is a single pipeline in a pipelin
			// it gets hoisted to the top inheriting the name of parent
			// as the name is likely reused elsewhere
			peek := originalNode.Pipeline.Children(RootNodeName)
			if len(peek) == 1 {
				for _, peekStage := range peek {
					if peekStage.Pipeline != nil {
						// aliased stage only contains a single item and that is a pipeline
						// move forward
						peekStage.DependsOn = clonedStage.DependsOn
						peekStage.Name = originalNode.Name
						originalNode = peekStage
						// originalNode.Pipeline = peekStage.Pipeline
					}
				}
			}
			// use alias or name
			// var aliasedStage *Stage
			for subNode := range originalNode.Pipeline.Nodes() {
				cloneHelper(originalNode.Pipeline, subNode, append(ancestralParentNames, originalNode.Name), flattenedStage)
			}
			clonedStage.Pipeline = subGraphClone
		}

		// Add the cloned node to the cloned graph
		// clonedGraph.nodes[uniqueName] = clonedStage
	}

	// Clone each child node, creating unique names based on the current path
	for _, child := range graph.Children(nodeName) {
		cloneHelper(graph, child.Name, ancestralParentNames, flattenedStage)
		// childUniqueName := cloneHelper(graph, child.Name, clonedGraph, ancestralParentNames, flattenedStage)
		// clonedGraph.children[uniqueName] = append(clonedGraph.children[uniqueName], childUniqueName)
		// clonedGraph.parent[childUniqueName] = append(clonedGraph.parent[childUniqueName], nodeName)
	}

	return uniqueName
}
