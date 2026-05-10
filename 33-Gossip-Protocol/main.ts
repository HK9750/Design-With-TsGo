type GossipValue = { value: string; version: number };

export class GossipNode {
  private readonly state = new Map<string, GossipValue>();

  constructor(readonly id: string) {}

  set(key: string, value: string): void {
    const current = this.state.get(key);
    this.state.set(key, { value, version: (current?.version ?? 0) + 1 });
  }

  gossipTo(peer: GossipNode): void {
    peer.merge(this.state);
    this.merge(peer.state);
  }

  get(key: string): string | null {
    return this.state.get(key)?.value ?? null;
  }

  private merge(remote: Map<string, GossipValue>): void {
    for (const [key, incoming] of remote) {
      const local = this.state.get(key);
      if (!local || incoming.version > local.version) this.state.set(key, { ...incoming });
    }
  }
}

function main(): void {
  const a = new GossipNode("a");
  const b = new GossipNode("b");
  a.set("leader", "a");
  a.gossipTo(b);
  console.log(b.get("leader"));
}

if (require.main === module) main();
