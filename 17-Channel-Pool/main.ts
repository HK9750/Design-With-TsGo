export class ResourcePool<T> {
  private readonly idle: T[] = [];
  private active = 0;

  constructor(private readonly max: number, private readonly factory: () => T, private readonly reset: (item: T) => void = () => {}) {}

  acquire(): T {
    const item = this.idle.pop();
    if (item) {
      this.active++;
      return item;
    }
    if (this.active >= this.max) throw new Error("pool exhausted");
    this.active++;
    return this.factory();
  }

  release(item: T): void {
    this.reset(item);
    this.active--;
    this.idle.push(item);
  }
}

function main(): void {
  const pool = new ResourcePool<string[]>(2, () => []);
  const channel = pool.acquire();
  channel.push("message");
  pool.release(channel);
  console.log("released");
}

if (require.main === module) main();
