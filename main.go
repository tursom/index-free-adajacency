package main

import (
	"fmt"
	"math/rand"
	"time"

	"index-free-adjacency/graph"
)

func init() {
	rand.Seed(time.Now().UnixMilli())
}

func main() {
	var g graph.Graph
	for j := 0; j < 16; j++ {
		for i := 0; i < 32; i++ {
			g.AddNode(fmt.Sprintf("test %d", i))
		}
		for i := 0; i < g.NodeCount(); i++ {
			_ = g.DeleteNode(rand.Int() % g.NodeCount())
		}
	}
	fmt.Println("check nodes struct", g.CheckNodes())
	nodes := g.Nodes()
	for nodes.HasNext() {
		node := nodes.Next()
		for i := 0; i < 16; i++ {
			node.SetProperty(fmt.Sprintf("test %d", i), fmt.Sprintf("test %d", uint8(rand.Int())))
		}

		for i := 0; i < 16; i++ {
			node.DelProperty(fmt.Sprintf("test %d", rand.Uint32()%16))
		}

		fmt.Println(node.ID(), node.Lable())
	}
	fmt.Println()
}
