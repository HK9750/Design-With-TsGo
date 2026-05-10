import { fnv1a_on_string } from "./fnv-hash";

type OAEntry<K, V> = {
  key: K;
  value: V;
  deleted?: boolean;
};

type ProbeResult = {
  index: number;
  found: boolean;
};

export class MyOAMap<K extends string, V> {
  private buckets: Array<OAEntry<K, V> | undefined>;
  private size: number;
  private used: number;
  private capacity: number;

  private readonly minimumThreshold = 0.25;
  private readonly maximumThreshold = 0.75;
  private readonly minimumCapacity = 16;

  constructor(capacity?: number) {
    this.capacity = Math.floor(capacity ?? this.minimumCapacity);
    if (this.capacity < this.minimumCapacity) {
      this.capacity = this.minimumCapacity;
    }

    this.size = 0;
    this.used = 0;
    this.buckets = new Array(this.capacity);
  }

  get length(): number {
    return this.size;
  }

  get bucketCount(): number {
    return this.capacity;
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

  private shouldGrowBeforeInsert(): boolean {
    return this.size + 1 > this.capacity * this.maximumThreshold;
  }

  private shouldCleanDeletedSlotsBeforeInsert(): boolean {
    return this.used + 1 > this.capacity * this.maximumThreshold;
  }

  private findSlot(key: K): ProbeResult {
    const start = this.hash(key);
    let firstDeleted = -1;

    for (let offset = 0; offset < this.capacity; offset++) {
      const index = (start + offset) % this.capacity;
      const entry = this.buckets[index];

      if (!entry) {
        return {
          index: firstDeleted === -1 ? index : firstDeleted,
          found: false,
        };
      }

      if (entry.deleted) {
        if (firstDeleted === -1) {
          firstDeleted = index;
        }
        continue;
      }

      if (entry.key === key) {
        return { index, found: true };
      }
    }

    return { index: firstDeleted, found: false };
  }

  private insertRehashedEntry(key: K, value: V): void {
    const slot = this.findSlot(key);
    if (slot.index === -1) {
      throw new Error("Hash table has no available slot during rehash");
    }

    this.buckets[slot.index] = { key, value };
    this.size++;
    this.used++;
  }

  get(key: K): V | null {
    const start = this.hash(key);

    for (let offset = 0; offset < this.capacity; offset++) {
      const index = (start + offset) % this.capacity;
      const entry = this.buckets[index];

      if (!entry) {
        return null;
      }

      if (!entry.deleted && entry.key === key) {
        return entry.value;
      }
    }

    return null;
  }

  set(key: K, value: V): void {
    let slot = this.findSlot(key);

    if (slot.found) {
      this.buckets[slot.index]!.value = value;
      return;
    }

    if (this.shouldGrowBeforeInsert()) {
      this.resize(this.capacity * 2);
      slot = this.findSlot(key);
    } else if (this.shouldCleanDeletedSlotsBeforeInsert()) {
      this.resize(this.capacity);
      slot = this.findSlot(key);
    }

    if (slot.index === -1) {
      this.resize(this.capacity * 2);
      slot = this.findSlot(key);
    }

    if (slot.index === -1) {
      throw new Error("Hash table has no available slot");
    }

    const previousEntry = this.buckets[slot.index];
    if (!previousEntry) {
      this.used++;
    }

    this.buckets[slot.index] = { key, value };
    this.size++;

    if (this.shouldGrow()) {
      this.resize(this.capacity * 2);
    }
  }

  delete(key: K): boolean {
    const start = this.hash(key);

    for (let offset = 0; offset < this.capacity; offset++) {
      const index = (start + offset) % this.capacity;
      const entry = this.buckets[index];

      if (!entry) {
        return false;
      }

      if (!entry.deleted && entry.key === key) {
        entry.deleted = true;
        this.size--;
        if (this.shouldShrink()) {
          this.resize(Math.floor(this.capacity / 2));
        }
        return true;
      }
    }

    return false;
  }

  private resize(newCapacity: number): void {
    if (newCapacity < this.minimumCapacity) {
      newCapacity = this.minimumCapacity;
    }

    const oldBuckets = this.buckets;
    this.capacity = Math.floor(newCapacity);
    this.size = 0;
    this.used = 0;
    this.buckets = new Array(this.capacity);

    for (const entry of oldBuckets) {
      if (entry && !entry.deleted) {
        this.insertRehashedEntry(entry.key, entry.value);
      }
    }
  }
}

function main(): void {
  const hashMap = new MyOAMap<string, number>();

  console.log("===== BASIC INSERT TEST =====");
  hashMap.set("one", 1);
  hashMap.set("two", 2);
  hashMap.set("three", 3);
  console.log("one:", hashMap.get("one"));
  console.log("two:", hashMap.get("two"));
  console.log("three:", hashMap.get("three"));
  console.log("four:", hashMap.get("four"));

  console.log("===== UPDATE TEST =====");
  hashMap.set("one", 100);
  console.log("one updated:", hashMap.get("one"));

  console.log("===== DELETE TEST =====");
  console.log("two deleted:", hashMap.delete("two"));
  console.log("two after delete:", hashMap.get("two"));

  console.log("===== TOMBSTONE REUSE TEST =====");
  hashMap.set("two", 222);
  console.log("two reinserted:", hashMap.get("two"));

  console.log("===== LARGE INSERT (GROW TEST) =====");
  const largeCount = 10_000;
  for (let i = 0; i < largeCount; i++) {
    hashMap.set(`key_${i}`, i);
  }

  let valid = true;
  for (let i = 0; i < largeCount; i++) {
    if (hashMap.get(`key_${i}`) !== i) {
      valid = false;
      break;
    }
  }
  console.log("Large insert test:", valid ? "PASSED" : "FAILED");

  console.log("===== DELETE MANY (SHRINK TEST) =====");
  for (let i = 0; i < largeCount - 100; i++) {
    hashMap.delete(`key_${i}`);
  }

  valid = true;
  for (let i = largeCount - 100; i < largeCount; i++) {
    if (hashMap.get(`key_${i}`) !== i) {
      valid = false;
      break;
    }
  }
  console.log("Shrink integrity test:", valid ? "PASSED" : "FAILED");

  console.log("===== EDGE CASES =====");
  hashMap.set("", 999);
  hashMap.set("0", 0);
  console.log("Empty string key:", hashMap.get(""));
  console.log("Zero value:", hashMap.get("0"));
  console.log("delete non-existent:", hashMap.delete("does_not_exist"));

  console.log("===== FINAL STATUS =====");
  console.log("Size:", hashMap.length, "Capacity:", hashMap.bucketCount);
}

if (require.main === module) {
  main();
}
