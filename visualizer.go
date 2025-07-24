package main

import (
	"fmt"
	"os"
	"strings"
)

type GraphNode struct {
	ID       string
	Label    string
	Type     string
	Location string
}

type GraphEdge struct {
	From     string
	To       string
	Label    string
	Location string
}

func generateGraph(channels map[string]*ChannelInfo) string {
	var nodes []GraphNode
	var edges []GraphEdge

	for name, channel := range channels {
		nodes = append(nodes, GraphNode{
			ID:       name,
			Label:    name,
			Type:     channel.Type,
			Location: channel.Location,
		})

		for _, sendOp := range channel.SendOps {
			edges = append(edges, GraphEdge{
				From:     "main",
				To:       name,
				Label:    "send",
				Location: sendOp,
			})
		}

		for _, receiveOp := range channel.ReceiveOps {
			edges = append(edges, GraphEdge{
				From:     name,
				To:       "main",
				Label:    "receive",
				Location: receiveOp,
			})
		}
	}

	var dot strings.Builder
	dot.WriteString("digraph ChannelFlow {\n")
	dot.WriteString("  rankdir=LR;\n")
	dot.WriteString("  node [shape=box, style=filled, fillcolor=lightblue];\n")
	dot.WriteString("  edge [color=gray];\n\n")

	for _, node := range nodes {
		dot.WriteString(fmt.Sprintf("  %s [label=\"%s\\n%s\\n%s\"];\n",
			node.ID, node.Label, node.Type, node.Location))
	}

	for _, edge := range edges {
		dot.WriteString(fmt.Sprintf("  %s -> %s [label=\"%s\\n%s\"];\n",
			edge.From, edge.To, edge.Label, edge.Location))
	}

	dot.WriteString("}\n")
	return dot.String()
}

func saveGraphToFile(dotContent string, filename string) error {
	return os.WriteFile(filename, []byte(dotContent), 0644)
}

func visualizeChannels(channels map[string]*ChannelInfo) {
	dotContent := generateGraph(channels)
	err := saveGraphToFile(dotContent, "channel_flow.dot")
	if err != nil {
		fmt.Printf("Error saving graph: %v\n", err)
		return
	}
	fmt.Println("\nGraph visualization saved to channel_flow.dot")
	fmt.Println("To view the graph, install Graphviz and run:")
	fmt.Println("dot -Tpng channel_flow.dot -o channel_flow.png")
} 