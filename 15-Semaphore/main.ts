export class Semaphore {
  private available: number;
  private readonly waiters: Array<() => void> = [];

  constructor(private readonly permits: number) {
    if (permits <= 0) throw new Error("permits must be positive");
    this.available = permits;
  }

  acquire(): Promise<() => void> {
    if (this.available > 0) {
      this.available--;
      return Promise.resolve(() => this.release());
    }
    return new Promise((resolve) => {
      this.waiters.push(() => resolve(() => this.release()));
    });
  }

  private release(): void {
    const waiter = this.waiters.shift();
    if (waiter) waiter();
    else this.available = Math.min(this.permits, this.available + 1);
  }
}

async function main(): Promise<void> {
  const semaphore = new Semaphore(1);
  const release = await semaphore.acquire();
  release();
  console.log("released");
}

if (require.main === module) void main();
