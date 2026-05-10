export class ShardedStore {
  private readonly shards: Array<Map<string, string>>;

  constructor(shardCount: number) {
    if (shardCount <= 0) throw new Error("shardCount must be positive");
    this.shards = Array.from({ length: shardCount }, () => new Map());
  }

  put(key: string, value: string): void {
    this.shardFor(key).set(key, value);
  }

  get(key: string): string | null {
    return this.shardFor(key).get(key) ?? null;
  }

  shardIndex(key: string): number {
    return hash32(key) % this.shards.length;
  }

  private shardFor(key: string): Map<string, string> {
    return this.shards[this.shardIndex(key)];
  }
}

function hash32(value: string): number {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i++) hash = Math.imul((hash ^ value.charCodeAt(i)) >>> 0, 16777619) >>> 0;
  return hash;
}

function main(): void {
  const store = new ShardedStore(4);
  store.put("user:1", "Ada");
  console.log(store.get("user:1"), store.shardIndex("user:1"));
}

if (require.main === module) main();
