export class PrimaryReplicaStore {
  private readonly primary = new Map<string, string>();
  private readonly replicas: Array<Map<string, string>>;

  constructor(replicaCount: number) {
    this.replicas = Array.from({ length: replicaCount }, () => new Map());
  }

  write(key: string, value: string): void {
    this.primary.set(key, value);
    for (const replica of this.replicas) replica.set(key, value);
  }

  read(key: string, fromReplica = false): string | null {
    const source = fromReplica ? this.replicas[0] : this.primary;
    return source?.get(key) ?? null;
  }

  failover(replicaIndex: number): void {
    const chosen = this.replicas[replicaIndex];
    if (!chosen) throw new Error("missing replica");
    this.primary.clear();
    for (const [key, value] of chosen) this.primary.set(key, value);
  }
}

function main(): void {
  const store = new PrimaryReplicaStore(2);
  store.write("x", "1");
  console.log(store.read("x", true));
}

if (require.main === module) main();
