export class BinaryHeap<T> {
  private data: T[] = [];

  constructor(private readonly higherPriority: (a: T, b: T) => boolean, values: T[] = []) {
    this.data = [...values];
    for (let i = Math.floor(this.data.length / 2) - 1; i >= 0; i--) this.siftDown(i);
  }

  get size(): number {
    return this.data.length;
  }

  peek(): T | null {
    return this.data[0] ?? null;
  }

  insert(value: T): void {
    this.data.push(value);
    this.siftUp(this.data.length - 1);
  }

  extract(): T | null {
    if (this.data.length === 0) return null;
    const top = this.data[0];
    const last = this.data.pop()!;
    if (this.data.length > 0) {
      this.data[0] = last;
      this.siftDown(0);
    }
    return top;
  }

  toArray(): T[] {
    return [...this.data];
  }

  private siftUp(index: number): void {
    while (index > 0) {
      const parent = Math.floor((index - 1) / 2);
      if (!this.higherPriority(this.data[index], this.data[parent])) break;
      [this.data[index], this.data[parent]] = [this.data[parent], this.data[index]];
      index = parent;
    }
  }

  private siftDown(index: number): void {
    for (;;) {
      const left = index * 2 + 1;
      const right = left + 1;
      let best = index;
      if (left < this.data.length && this.higherPriority(this.data[left], this.data[best])) best = left;
      if (right < this.data.length && this.higherPriority(this.data[right], this.data[best])) best = right;
      if (best === index) return;
      [this.data[index], this.data[best]] = [this.data[best], this.data[index]];
      index = best;
    }
  }
}

function main(): void {
  const minHeap = new BinaryHeap<number>((a, b) => a < b, [5, 1, 3]);
  minHeap.insert(2);
  console.log(minHeap.extract(), minHeap.extract());
}

if (require.main === module) main();
