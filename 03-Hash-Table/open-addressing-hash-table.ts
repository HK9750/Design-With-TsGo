import { fnv1a_on_string } from "./fnv-hash";

type OAEntry<K, V> = {
  key: K;
  value: V;
  deleted?: boolean;
};

export class MyOAMap<K extends string, V> {
  private buckets: Array<OAEntry<K, V> | undefined>;
  private size: number;
  private capacity: number;

  private readonly minimumThreshold = 0.25;
  private readonly maximumThreshold = 0.75;
  private readonly minimumCapacity = 16;

  constructor(capacity?: number) {
    this.capacity = capacity ?? this.minimumCapacity;
    if (this.capacity < this.minimumCapacity) {
      this.capacity = this.minimumCapacity;
    }

    this.size = 0;
    this.buckets = new Array(this.capacity);
  }

  private hash(key: K): number {
    const hash32 = Number(fnv1a_on_string(key, 32));
    return hash32 % this.capacity;
  }

  private shouldGrow(): boolean {
    return this.size > this.capacity * this.maximumThreshold;
  }

  private shouldShrink(): boolean {
    return (
      this.capacity > this.minimumCapacity &&
      this.size < this.capacity * this.minimumThreshold
    );
  }

  get(key: K): V | null {
    const index = this.hash(key);
    let i = index;
    let entry = this.buckets[i];

    while (entry) {
      if (!entry.deleted && entry.key === key) {
        return entry.value;
      }
      i = (i + 1) % this.capacity;
      if (i === index) {
        break;
      }
      entry = this.buckets[i];
    }

    return null;
  }

  set(key: K, value: V): void {
    const index = this.hash(key);
    let i = index;
    let entry = this.buckets[i];

    while (entry) {
      if (!entry.deleted && entry.key === key) {
        entry.value = value;
        return;
      }
      i = (i + 1) % this.capacity;
      if (i === index) {
        break;
      }
      entry = this.buckets[i];
    }

    this.buckets[i] = { key, value };
    this.size++;

    if (this.shouldGrow()) {
      this.resize(this.capacity * 2);
    }
  }

  delete(key: K): void {
    const index = this.hash(key);
    let i = index;
    let entry = this.buckets[i];

    while (entry) {
      if (!entry.deleted && entry.key === key) {
        entry.deleted = true;
        this.size--;
        if (this.shouldShrink()) {
          this.resize(Math.floor(this.capacity / 2));
        }
        return;
      }
      i = (i + 1) % this.capacity;
      if (i === index) {
        break;
      }
      entry = this.buckets[i];
    }
  }

  private resize(newCapacity: number): void {
    const oldBuckets = this.buckets;
    this.capacity = newCapacity;
    this.size = 0;
    this.buckets = new Array(this.capacity);

    for (const entry of oldBuckets) {
      if (entry && !entry.deleted) {
        this.set(entry.key, entry.value);
      }
    }
  }
}
