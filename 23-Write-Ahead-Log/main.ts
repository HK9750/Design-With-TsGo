export type WALEntry = { lsn: number; op: "put" | "delete"; key: string; value?: string; checksum: number };

export class WriteAheadLog {
  private readonly entries: WALEntry[] = [];
  private nextLsn = 1;

  append(op: "put" | "delete", key: string, value = ""): WALEntry {
    const base = `${this.nextLsn}|${op}|${key}|${value}`;
    const entry = { lsn: this.nextLsn++, op, key, value, checksum: checksum(base) };
    this.entries.push(entry);
    return entry;
  }

  recover(): Map<string, string> {
    const state = new Map<string, string>();
    for (const entry of this.entries) {
      const base = `${entry.lsn}|${entry.op}|${entry.key}|${entry.value ?? ""}`;
      if (entry.checksum !== checksum(base)) throw new Error(`corrupt WAL entry ${entry.lsn}`);
      if (entry.op === "put") state.set(entry.key, entry.value ?? "");
      else state.delete(entry.key);
    }
    return state;
  }
}

function checksum(value: string): number {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i++) hash = Math.imul((hash ^ value.charCodeAt(i)) >>> 0, 16777619) >>> 0;
  return hash;
}

function main(): void {
  const wal = new WriteAheadLog();
  wal.append("put", "x", "1");
  wal.append("delete", "x");
  console.log(wal.recover().has("x"));
}

if (require.main === module) main();
