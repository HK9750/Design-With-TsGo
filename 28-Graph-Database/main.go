package main

import (
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// GraphDatabase is an in-memory graph database supporting nodes with string
// properties and directed/undirected edges. It provides shortest path
// computation via BFS.
// Space complexity: O(V + E) where V is the number of vertices and E is edges.
type GraphDatabase struct {
	nodes map[string]map[string]string
	edges map[string]map[string]bool
}

// NewGraphDatabase creates an empty graph database with no nodes or edges.
// Time complexity: O(1).
func NewGraphDatabase() *GraphDatabase {
	log.Debug("Created GraphDatabase")
	return &GraphDatabase{
		nodes: make(map[string]map[string]string),
		edges: make(map[string]map[string]bool),
	}
}

// AddNode creates or updates a node with the given ID and property map.
// Automatically initializes an empty adjacency set if the node is new.
// Time complexity: O(1).
func (g *GraphDatabase) AddNode(id string, properties map[string]string) {
	defer log.Operation("GraphDatabase.AddNode", "id=%s props=%v", id, properties)()
	g.nodes[id] = properties
	if g.edges[id] == nil {
		g.edges[id] = make(map[string]bool)
		log.Debug("Initialized adjacency list for %s", id)
	}
}

// AddEdge creates an edge between two nodes. If either node does not exist,
// it is auto-created with empty properties. If directed is false, a reciprocal
// edge is also added, making the relationship bidirectional.
// Time complexity: O(1).
func (g *GraphDatabase) AddEdge(from, to string, directed bool) {
	defer log.Operation("GraphDatabase.AddEdge", "from=%s to=%s directed=%v", from, to, directed)()
	if g.nodes[from] == nil {
		g.AddNode(from, map[string]string{})
		log.Debug("Auto-created node: %s", from)
	}
	if g.nodes[to] == nil {
		g.AddNode(to, map[string]string{})
		log.Debug("Auto-created node: %s", to)
	}
	g.edges[from][to] = true
	if !directed {
		g.edges[to][from] = true
	}
	log.Debug("Edge added: %s -> %s", from, to)
}

// ShortestPath finds the shortest path between two nodes using Breadth-First
// Search (BFS) on an unweighted graph. Returns the sequence of node IDs from
// start to end, or nil if no path exists.
// Time complexity: O(V + E).
func (g *GraphDatabase) ShortestPath(start, end string) []string {
	defer log.Operation("GraphDatabase.ShortestPath", "start=%s end=%s", start, end)()
	if g.nodes[start] == nil {
		log.Warn("Start node %q does not exist", start)
		return nil
	}
	if g.nodes[end] == nil {
		log.Warn("End node %q does not exist", end)
		return nil
	}
	queue := [][]string{{start}}
	seen := map[string]bool{start: true}
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		node := path[len(path)-1]
		if node == end {
			log.Info("Shortest path found: length=%d, nodes=%v", len(path), path)
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
	log.Warn("No path found from %q to %q", start, end)
	return nil
}

func main() {
	defer log.Operation("main", "Running Graph Database demo")()
	defer log.Info("Graph Database demo completed")

	logger.Section("Graph Database — Social Network Analysis")
	log.Info("Simulating a social graph with user nodes and friendship edges")
	log.Info("Nodes have properties (name, city); edges represent mutual friendships")

	graph := NewGraphDatabase()

	logger.Section("Building the Social Graph")
	start := time.Now()
	graph.AddNode("alice", map[string]string{"name": "Alice", "city": "London"})
	graph.AddNode("bob", map[string]string{"name": "Bob", "city": "Paris"})
	graph.AddNode("charlie", map[string]string{"name": "Charlie", "city": "Berlin"})
	graph.AddNode("diana", map[string]string{"name": "Diana", "city": "Tokyo"})
	graph.AddNode("eve", map[string]string{"name": "Eve", "city": "New York"})
	graph.AddNode("frank", map[string]string{"name": "Frank", "city": "Sydney"})

	// Friendship network:
	// alice - bob - charlie - diana
	//   |               |
	//   +---- eve ------+
	// frank (isolated)
	graph.AddEdge("alice", "bob", false)
	graph.AddEdge("bob", "charlie", false)
	graph.AddEdge("charlie", "diana", false)
	graph.AddEdge("alice", "eve", false)
	graph.AddEdge("eve", "charlie", false)
	log.Info("Built graph with %d nodes and friendships in %v", 6, time.Since(start))

	logger.Section("Query 1 — Shortest Path: Direct Friends")
	path := graph.ShortestPath("alice", "bob")
	if path != nil {
		log.Info("Alice -> Bob: %v (1 hop)", path)
	}

	logger.Section("Query 2 — Shortest Path: Friend-of-Friend")
	path = graph.ShortestPath("alice", "diana")
	if path != nil {
		log.Info("Alice -> Diana: %v (3 hops via %s)", path, path[1])
	}

	logger.Section("Query 3 — Alternative Path (Through Eve)")
	path = graph.ShortestPath("alice", "charlie")
	if path != nil {
		log.Info("Alice -> Charlie: %v (2 hops)", path)
	}

	logger.Section("Query 4 — Isolated Node")
	path = graph.ShortestPath("alice", "frank")
	if path == nil {
		log.Warn("Alice -> Frank: No path exists (Frank is isolated)")
	}

	logger.Section("Query 5 — Non-Existent Node")
	path = graph.ShortestPath("alice", "zoe")
	if path == nil {
		log.Warn("Alice -> Zoe: No path — Zoe does not exist in the graph")
	}

	logger.Section("Stats Summary")
	logger.KeyValue("total_nodes", len(graph.nodes))
	edgeCount := 0
	for _, adj := range graph.edges {
		edgeCount += len(adj)
	}
	logger.KeyValue("total_edges", edgeCount/2)
	logger.KeyValue("isolated_nodes", 1)
}
