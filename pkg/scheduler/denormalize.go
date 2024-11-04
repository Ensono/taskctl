package scheduler

import (
	"fmt"
	"strings"

	"github.com/Ensono/taskctl/internal/utils"
)

// DenormalizePipeline performs a DFS traversal on the ExecutionGraph from the root node
//
// In order to be able to call the same pipeline from another pipeline, we want to create a new
// pointer to it, this will avoid race conditions in times/outputs/env vars/etc...
// We can also set separate vars and environment variables
func (g *ExecutionGraph) DenormalizePipeline() (*ExecutionGraph, error) {
	denormalizedGraph, _ := NewExecutionGraph(g.Name())
	flattenedStages := map[string]*Stage{}

	cloneHelper(g, RootNodeName, []string{g.Name()}, flattenedStages)
	// rebuild graph from flatten denormalized stages
	denormalizedGraph.rebuildFromDenormalized(StageTable(flattenedStages))
	return denormalizedGraph, nil
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

func (g *ExecutionGraph) rebuildFromDenormalized(st StageTable) error {
	for _, stage := range st.FirstLevelChildren(g.Name(), 1) {
		if stage.Pipeline != nil {
			c := st.FirstLevelChildren(stage.Name, 1)
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

// StageTable is a simple hash table of denormalized stages into a flat hash table (map)
//
// NOTE: used for read only at this point
type StageTable map[string]*Stage

// FirstLevelChildren retrieves the nodes by prefix and depth
func (st StageTable) FirstLevelChildren(prefix string, depth int) []*Stage {
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

// cloneHelper is a recursive helper function to clone nodes with unique paths
// each new instance will have a separate memory address allocation
func cloneHelper(graph *ExecutionGraph, nodeName string, ancestralParentNames []string, flattenedStage map[string]*Stage) string {
	uniqueName := utils.CascadeName(ancestralParentNames, nodeName)
	if nodeName != RootNodeName {
		// Clone the node - dereferencing
		originalNode, _ := graph.Node(nodeName)
		clonedStage := NewStage(uniqueName)
		// Task or stage needs adding
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
				cloneHelper(originalNode.Pipeline, subNode, append(ancestralParentNames, originalNode.Name), flattenedStage)
			}
			clonedStage.Pipeline = subGraphClone
		}
	}

	// Clone each child node, creating unique names based on the current path
	for _, child := range graph.Children(nodeName) {
		cloneHelper(graph, child.Name, ancestralParentNames, flattenedStage)
	}

	return uniqueName
}
