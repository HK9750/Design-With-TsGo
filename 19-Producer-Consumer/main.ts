export class BoundedQueue<T> {
  private readonly queue: T[] = [];
  private readonly producers: Array<() => void> = [];
  private readonly consumers: Array<(value: T) => void> = [];

  constructor(private readonly capacity: number) {}

  async enqueue(value: T): Promise<void> {
    if (this.consumers.length > 0) {
      this.consumers.shift()!(value);
      return;
    }
    while (this.queue.length >= this.capacity) await new Promise<void>((resolve) => this.producers.push(resolve));
    this.queue.push(value);
  }

  dequeue(): Promise<T> {
    const value = this.queue.shift();
    if (value !== undefined) {
      this.producers.shift()?.();
      return Promise.resolve(value);
    }
    return new Promise((resolve) => this.consumers.push(resolve));
  }
}

async function main(): Promise<void> {
  const queue = new BoundedQueue<number>(1);
  await queue.enqueue(42);
  console.log(await queue.dequeue());
}

if (require.main === module) void main();
