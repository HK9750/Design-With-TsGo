type Role = "leader" | "follower";

export class BullyElectionCluster {
  private readonly alive = new Set<number>();
  private readonly roles = new Map<number, Role>();
  private leaderId: number | null = null;

  constructor(nodeIds: number[]) {
    for (const id of nodeIds) {
      this.alive.add(id);
      this.roles.set(id, "follower");
    }
    this.elect();
  }

  fail(nodeId: number): void {
    this.alive.delete(nodeId);
    if (this.leaderId === nodeId) this.elect();
  }

  recover(nodeId: number): void {
    this.alive.add(nodeId);
    this.elect();
  }

  elect(): number | null {
    this.leaderId = [...this.alive].sort((a, b) => b - a)[0] ?? null;
    for (const id of this.roles.keys()) this.roles.set(id, id === this.leaderId ? "leader" : "follower");
    return this.leaderId;
  }

  leader(): number | null {
    return this.leaderId;
  }
}

function main(): void {
  const cluster = new BullyElectionCluster([1, 2, 3]);
  cluster.fail(3);
  console.log(cluster.leader());
}

if (require.main === module) main();
