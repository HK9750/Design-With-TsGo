type RaftRole = "follower" | "candidate" | "leader";
type LogEntry = { term: number; command: string };

class RaftNode {
  role: RaftRole = "follower";
  term = 0;
  votedFor: number | null = null;
  log: LogEntry[] = [];
  commitIndex = -1;
  constructor(readonly id: number) {}
}

export class RaftCluster {
  private readonly nodes: RaftNode[];
  private leaderId: number | null = null;

  constructor(nodeIds: number[]) {
    this.nodes = nodeIds.map((id) => new RaftNode(id));
  }

  elect(candidateId: number): boolean {
    const candidate = this.node(candidateId);
    candidate.role = "candidate";
    candidate.term++;
    candidate.votedFor = candidate.id;
    let votes = 1;
    for (const peer of this.nodes.filter((n) => n.id !== candidateId)) {
      if (candidate.term >= peer.term && (peer.votedFor === null || peer.votedFor === candidate.id)) {
        peer.term = candidate.term;
        peer.votedFor = candidate.id;
        votes++;
      }
    }
    if (votes > this.nodes.length / 2) {
      this.leaderId = candidate.id;
      for (const node of this.nodes) node.role = node.id === candidate.id ? "leader" : "follower";
      return true;
    }
    return false;
  }

  append(command: string): boolean {
    if (this.leaderId === null) return false;
    const leader = this.node(this.leaderId);
    const entry = { term: leader.term, command };
    leader.log.push(entry);
    let replicated = 1;
    for (const peer of this.nodes.filter((n) => n.id !== leader.id)) {
      peer.log.push(entry);
      replicated++;
    }
    if (replicated > this.nodes.length / 2) {
      const index = leader.log.length - 1;
      for (const node of this.nodes) node.commitIndex = index;
      return true;
    }
    return false;
  }

  committedCommands(): string[] {
    if (this.leaderId === null) return [];
    const leader = this.node(this.leaderId);
    return leader.log.slice(0, leader.commitIndex + 1).map((entry) => entry.command);
  }

  private node(id: number): RaftNode {
    const node = this.nodes.find((n) => n.id === id);
    if (!node) throw new Error("missing node");
    return node;
  }
}

function main(): void {
  const cluster = new RaftCluster([1, 2, 3]);
  cluster.elect(1);
  cluster.append("set x=1");
  console.log(cluster.committedCommands());
}

if (require.main === module) main();
