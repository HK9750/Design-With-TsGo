type KVEntry = { value: string; expiresAt: number | null; version: number };

export class KeyValueStore {
  private readonly data = new Map<string, KVEntry>();
  private version = 0;

  put(key: string, value: string, ttlMs?: number): number {
    const expiresAt = ttlMs === undefined ? null : Date.now() + ttlMs;
    const version = ++this.version;
    this.data.set(key, { value, expiresAt, version });
    return version;
  }

  get(key: string): string | null {
    const entry = this.data.get(key);
    if (!entry) return null;
    if (entry.expiresAt !== null && entry.expiresAt <= Date.now()) {
      this.data.delete(key);
      return null;
    }
    return entry.value;
  }

  delete(key: string): boolean {
    return this.data.delete(key);
  }

  range(start: string, end: string): Array<[string, string]> {
    return [...this.data.keys()].sort().filter((key) => key >= start && key <= end).map((key) => [key, this.get(key)] as [string, string | null]).filter((entry): entry is [string, string] => entry[1] !== null);
  }

  snapshot(): Map<string, string> {
    return new Map([...this.data.keys()].map((key) => [key, this.get(key)] as [string, string | null]).filter((entry): entry is [string, string] => entry[1] !== null));
  }
}

function main(): void {
  const store = new KeyValueStore();
  store.put("a", "1");
  store.put("b", "2");
  console.log(store.range("a", "z"));
}

if (require.main === module) main();
