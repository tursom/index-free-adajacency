package graph

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixMilli())
}

func TestGraph_AddRelation(t *testing.T) {
	var g Graph
	for i := 0; i < 1024; i++ {
		g.AddNode(fmt.Sprintf("node %d", i))
	}

	for j := 0; j < 16; j++ {
		for i := 0; i < 1024; i++ {
			id := g.AddRelation(i, rand.Int()%g.NodeCount())
			relation := g.GetRelation(id)
			relation.SetProperty(fmt.Sprintf("hello %d", relation.From().ID()),
				fmt.Sprintf("world %d", relation.To().ID()))
		}
		_ = g.DeleteRelation(rand.Int() % g.RelationCount())
	}

	Loop(g.Nodes(), func(node *Node) {
		Loop(node.Relations(), func(relation *Relation) {
			fmt.Println(relation)
		})
	})

	fmt.Println(g.CheckRelations())
}
