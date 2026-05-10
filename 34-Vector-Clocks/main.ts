export type ClockOrder = "before" | "after" | "equal" | "concurrent";

export class VectorClock {
  private readonly clock = new Map<string, number>();

  tick(node: string): void {
    this.clock.set(node, (this.clock.get(node) ?? 0) + 1);
  }

  merge(other: VectorClock): void {
    for (const [node, value] of other.clock) this.clock.set(node, Math.max(this.clock.get(node) ?? 0, value));
  }

  compare(other: VectorClock): ClockOrder {
    let less = false;
    let greater = false;
    for (const node of new Set([...this.clock.keys(), ...other.clock.keys()])) {
      const a = this.clock.get(node) ?? 0;
      const b = other.clock.get(node) ?? 0;
      if (a < b) less = true;
      if (a > b) greater = true;
    }
    if (less && greater) return "concurrent";
    if (less) return "before";
    if (greater) return "after";
    return "equal";
  }
}

function main(): void {
  const a = new VectorClock();
  const b = new VectorClock();
  a.tick("a");
  b.tick("b");
  console.log(a.compare(b));
}

if (require.main === module) main();
