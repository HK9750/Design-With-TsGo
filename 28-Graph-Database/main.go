package main

import "fmt"

type GraphDatabase struct {
	nodes map[string]map[string]string
	edges map[string]map[string]bool
}

func NewGraphDatabase() *GraphDatabase {
	return &GraphDatabase{nodes: make(map[string]map[string]string), edges: make(map[string]map[string]bool)}
}

func (g *GraphDatabase) AddNode(id string, properties map[string]string) {
	g.nodes[id] = properties
	if g.edges[id] == nil {
		g.edges[id] = make(map[string]bool)
	}
}

func (g *GraphDatabase) AddEdge(from, to string, directed bool) {
	if g.nodes[from] == nil {
		g.AddNode(from, map[string]string{})
	}
	if g.nodes[to] == nil {
		g.AddNode(to, map[string]string{})
	}
	g.edges[from][to] = true
	if !directed {
		g.edges[to][from] = true
	}
}

func (g *GraphDatabase) ShortestPath(start, end string) []string {
	queue := [][]string{{start}}
	seen := map[string]bool{start: true}
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		node := path[len(path)-1]
		if node == end {
			return path
		}
		for next := range g.edges[node] {
			if !seen[next] {
				seen[next] = true
				copyPath := append(append([]string{}, path...), next)
				queue = append(queue, copyPath)
			}
		}
	}
	return nil
}

func main() {
	graph := NewGraphDatabase()
	graph.AddEdge("a", "b", false)
	graph.AddEdge("b", "c", false)
	fmt.Println(graph.ShortestPath("a", "c"))
}
