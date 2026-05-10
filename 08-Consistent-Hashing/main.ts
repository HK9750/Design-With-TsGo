type RingPoint = { hash: number; node: string };

export class ConsistentHashRing {
  private readonly ring: RingPoint[] = [];

  constructor(private readonly replicas = 100) {}

  addNode(node: string): void {
    this.removeNode(node);
    for (let i = 0; i < this.replicas; i++) this.ring.push({ hash: hash32(`${node}#${i}`), node });
    this.ring.sort((a, b) => a.hash - b.hash);
  }

  removeNode(node: string): void {
    for (let i = this.ring.length - 1; i >= 0; i--) {
      if (this.ring[i].node === node) this.ring.splice(i, 1);
    }
  }

  getNode(key: string): string | null {
    if (this.ring.length === 0) return null;
    const hash = hash32(key);
    let low = 0;
    let high = this.ring.length - 1;
    while (low <= high) {
      const mid = Math.floor((low + high) / 2);
      if (this.ring[mid].hash < hash) low = mid + 1;
      else high = mid - 1;
    }
    return this.ring[low % this.ring.length].node;
  }
}

function hash32(value: string): number {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 16777619) >>> 0;
  }
  return hash;
}

function main(): void {
  const ring = new ConsistentHashRing(50);
  ring.addNode("node-a");
  ring.addNode("node-b");
  console.log(ring.getNode("customer:42"));
}

if (require.main === module) main();
