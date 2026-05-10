export class AsyncReadWriteLock {
  private readers = 0;
  private writer = false;
  private waitingWriters = 0;
  private readonly queue: Array<() => void> = [];

  acquireRead(): Promise<() => void> {
    if (!this.writer && this.waitingWriters === 0) {
      this.readers++;
      return Promise.resolve(() => this.releaseRead());
    }
    return new Promise((resolve) => this.queue.push(() => {
      this.readers++;
      resolve(() => this.releaseRead());
    }));
  }

  acquireWrite(): Promise<() => void> {
    this.waitingWriters++;
    if (!this.writer && this.readers === 0) {
      this.waitingWriters--;
      this.writer = true;
      return Promise.resolve(() => this.releaseWrite());
    }
    return new Promise((resolve) => this.queue.push(() => {
      this.waitingWriters--;
      this.writer = true;
      resolve(() => this.releaseWrite());
    }));
  }

  private releaseRead(): void {
    this.readers--;
    this.drain();
  }

  private releaseWrite(): void {
    this.writer = false;
    this.drain();
  }

  private drain(): void {
    if (this.writer || this.readers > 0 || this.queue.length === 0) return;
    this.queue.shift()!();
  }
}

async function main(): Promise<void> {
  const lock = new AsyncReadWriteLock();
  const release = await lock.acquireRead();
  release();
  console.log("read complete");
}

if (require.main === module) void main();
