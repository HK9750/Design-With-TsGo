export class GraphDatabase {
  private readonly nodes = new Map<string, Record<string, string>>();
  private readonly edges = new Map<string, Set<string>>();

  addNode(id: string, properties: Record<string, string> = {}): void {
    this.nodes.set(id, properties);
    if (!this.edges.has(id)) this.edges.set(id, new Set());
  }

  addEdge(from: string, to: string, directed = true): void {
    if (!this.nodes.has(from)) this.addNode(from);
    if (!this.nodes.has(to)) this.addNode(to);
    this.edges.get(from)!.add(to);
    if (!directed) this.edges.get(to)!.add(from);
  }

  neighbors(id: string): string[] {
    return [...(this.edges.get(id) ?? [])];
  }

  shortestPath(start: string, end: string): string[] | null {
    const queue: string[][] = [[start]];
    const seen = new Set([start]);
    while (queue.length > 0) {
      const path = queue.shift()!;
      const node = path[path.length - 1];
      if (node === end) return path;
      for (const next of this.neighbors(node)) if (!seen.has(next)) {
        seen.add(next);
        queue.push([...path, next]);
      }
    }
    return null;
  }
}

function main(): void {
  const graph = new GraphDatabase();
  graph.addEdge("a", "b", false);
  graph.addEdge("b", "c", false);
  console.log(graph.shortestPath("a", "c"));
}

if (require.main === module) main();
