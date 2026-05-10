export class GCounter {
  private readonly counts = new Map<string, number>();

  increment(node: string, amount = 1): void {
    if (amount < 0) throw new Error("GCounter cannot decrement");
    this.counts.set(node, (this.counts.get(node) ?? 0) + amount);
  }

  merge(other: GCounter): void {
    for (const [node, count] of other.counts) this.counts.set(node, Math.max(this.counts.get(node) ?? 0, count));
  }

  value(): number {
    return [...this.counts.values()].reduce((sum, count) => sum + count, 0);
  }
}

export class PNCounter {
  private readonly positive = new GCounter();
  private readonly negative = new GCounter();

  increment(node: string, amount = 1): void { this.positive.increment(node, amount); }
  decrement(node: string, amount = 1): void { this.negative.increment(node, amount); }
  merge(other: PNCounter): void { this.positive.merge(other.positive); this.negative.merge(other.negative); }
  value(): number { return this.positive.value() - this.negative.value(); }
}

function main(): void {
  const a = new PNCounter();
  const b = new PNCounter();
  a.increment("a", 3);
  b.decrement("b", 1);
  a.merge(b);
  console.log(a.value());
}

if (require.main === module) main();
