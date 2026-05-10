type LSMValue = { value: string | null; deleted: boolean; sequence: number };

export class LSMTree {
  private readonly memtable = new Map<string, LSMValue>();
  private readonly sstables: Array<Map<string, LSMValue>> = [];
  private sequence = 0;

  constructor(private readonly flushThreshold = 4) {}

  put(key: string, value: string): void {
    this.memtable.set(key, { value, deleted: false, sequence: ++this.sequence });
    if (this.memtable.size >= this.flushThreshold) this.flush();
  }

  delete(key: string): void {
    this.memtable.set(key, { value: null, deleted: true, sequence: ++this.sequence });
    if (this.memtable.size >= this.flushThreshold) this.flush();
  }

  get(key: string): string | null {
    const live = this.memtable.get(key);
    if (live) return live.deleted ? null : live.value;
    for (let i = this.sstables.length - 1; i >= 0; i--) {
      const entry = this.sstables[i].get(key);
      if (entry) return entry.deleted ? null : entry.value;
    }
    return null;
  }

  flush(): void {
    if (this.memtable.size === 0) return;
    this.sstables.push(new Map([...this.memtable.entries()].sort(([a], [b]) => a.localeCompare(b))));
    this.memtable.clear();
  }

  compact(): void {
    const merged = new Map<string, LSMValue>();
    for (const table of this.sstables) {
      for (const [key, entry] of table) {
        const current = merged.get(key);
        if (!current || entry.sequence > current.sequence) merged.set(key, entry);
      }
    }
    for (const [key, entry] of [...merged]) if (entry.deleted) merged.delete(key);
    this.sstables.splice(0, this.sstables.length, new Map([...merged.entries()].sort(([a], [b]) => a.localeCompare(b))));
  }
}

function main(): void {
  const tree = new LSMTree(2);
  tree.put("a", "1");
  tree.put("b", "2");
  tree.delete("a");
  console.log(tree.get("a"), tree.get("b"));
}

if (require.main === module) main();
