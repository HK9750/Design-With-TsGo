// it is a simple implementation of how a hash table works at the low level.. how is read,insert,update O(1) is achieved..

// A hashtable is basically a key value pair which uses a hash function to hash the key to an index..
// we are using fnv1a hash function for this.. i dont know what fnv1a is but it looked nice xd.

// A hash table consists of the following things:
//    - Array<Bucket> -> a bucket can be a linkedList(chaining) or openaddressing slot
//    - size
//    - capacity
//    - loadfactorthreshold i.e we will have a 0.75 ratio for maximum and 0.25 for minimum but start with 16.
//    - hashfunction - fnv1a in this case

// loadfactor is basically the ratio between the size/capacity
// if loadfactor > loadfactorthreshold then we will resize the hash table to maintain performance

import { fnv1a_on_string } from "./fnv-hash";

type Entry<K, V> = {
  key: K;
  value: V;
  next?: Entry<K, V>;
};

export class MyMap<K extends string, V> {
  private buckets: Array<Entry<K, V> | undefined>;
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
    let entry = this.buckets[index];

    while (entry) {
      if (entry.key === key) {
        return entry.value;
      }
      entry = entry.next;
    }

    return null;
  }

  set(key: K, value: V): void {
    const index = this.hash(key);
    let entry = this.buckets[index];

    while (entry) {
      if (entry.key === key) {
        entry.value = value;
        return;
      }
      entry = entry.next;
    }

    const newEntry: Entry<K, V> = {
      key,
      value,
      next: this.buckets[index],
    };

    this.buckets[index] = newEntry;
    this.size++;

    if (this.shouldGrow()) {
      this.resize(this.capacity * 2);
    }
  }

  delete(key: K): boolean {
    const index = this.hash(key);
    let entry = this.buckets[index];
    let prev: Entry<K, V> | undefined;

    while (entry) {
      if (entry.key === key) {
        if (prev) {
          prev.next = entry.next;
        } else {
          this.buckets[index] = entry.next;
        }

        this.size--;

        if (this.shouldShrink()) {
          this.resize(Math.floor(this.capacity / 2));
        }

        return true;
      }

      prev = entry;
      entry = entry.next;
    }

    return false;
  }

  private resize(newCapacity: number): void {
    if (newCapacity < this.minimumCapacity) {
      newCapacity = this.minimumCapacity;
    }

    const oldBuckets = this.buckets;

    this.capacity = newCapacity;
    this.buckets = new Array(this.capacity);
    this.size = 0;

    for (const bucket of oldBuckets) {
      let entry = bucket;

      while (entry) {
        const index = this.hash(entry.key);

        const newEntry: Entry<K, V> = {
          key: entry.key,
          value: entry.value,
          next: this.buckets[index],
        };

        this.buckets[index] = newEntry;
        this.size++;

        entry = entry.next;
      }
    }
  }
}

function main() {
  const hashMap = new MyMap<string, number>();

  console.log("===== BASIC INSERT TEST =====");
  hashMap.set("one", 1);
  hashMap.set("two", 2);
  hashMap.set("three", 3);

  console.log(hashMap.get("one"));
  console.log(hashMap.get("two"));
  console.log(hashMap.get("three"));
  console.log(hashMap.get("four"));

  console.log("===== UPDATE TEST =====");
  hashMap.set("one", 100);
  console.log(hashMap.get("one"));

  console.log("===== DELETE TEST =====");
  hashMap.delete("two");
  console.log(hashMap.get("two"));

  console.log("===== RE-INSERT AFTER DELETE =====");
  hashMap.set("two", 222);
  console.log(hashMap.get("two"));

  console.log("===== COLLISION TEST =====");
  for (let i = 0; i < 50; i++) {
    hashMap.set("collision_" + i, i);
  }

  for (let i = 0; i < 50; i++) {
    if (hashMap.get("collision_" + i) !== i) {
      console.error("Collision retrieval failed at", i);
    }
  }

  console.log("Collision test passed");

  console.log("===== LARGE INSERT (GROW TEST) =====");
  const largeCount = 10000;

  for (let i = 0; i < largeCount; i++) {
    hashMap.set("key_" + i, i);
  }

  let valid = true;
  for (let i = 0; i < largeCount; i++) {
    if (hashMap.get("key_" + i) !== i) {
      valid = false;
      console.error("Mismatch at key_", i);
      break;
    }
  }

  console.log("Large insert test:", valid ? "PASSED" : "FAILED");

  console.log("===== DELETE MANY (SHRINK TEST) =====");
  for (let i = 0; i < largeCount - 100; i++) {
    hashMap.delete("key_" + i);
  }

  valid = true;
  for (let i = largeCount - 100; i < largeCount; i++) {
    if (hashMap.get("key_" + i) !== i) {
      valid = false;
      console.error("Post-shrink mismatch at key_", i);
      break;
    }
  }

  console.log("Shrink integrity test:", valid ? "PASSED" : "FAILED");

  console.log("===== DELETE NON-EXISTENT =====");
  console.log(hashMap.delete("does_not_exist"));

  console.log("===== RANDOM STRESS TEST =====");
  const randomKeys: string[] = [];

  for (let i = 0; i < 5000; i++) {
    const key = "rand_" + Math.floor(Math.random() * 1_000_000);
    randomKeys.push(key);
    hashMap.set(key, i);
  }

  let randomValid = true;
  for (let i = 0; i < randomKeys.length; i++) {
    if (hashMap.get(randomKeys[i]) === null) {
      randomValid = false;
      console.error("Random retrieval failed at", randomKeys[i]);
      break;
    }
  }

  console.log("Random stress test:", randomValid ? "PASSED" : "FAILED");

  console.log("===== DELETE ALL TEST =====");
  for (const key of randomKeys) {
    hashMap.delete(key);
  }

  let allDeleted = true;
  for (const key of randomKeys) {
    if (hashMap.get(key) !== null) {
      allDeleted = false;
      console.error("Delete all failed at", key);
      break;
    }
  }

  console.log("Delete all test:", allDeleted ? "PASSED" : "FAILED");

  console.log("===== EDGE CASES =====");
  hashMap.set("", 999);
  console.log("Empty string key:", hashMap.get(""));

  hashMap.set("0", 0);
  console.log("Zero value:", hashMap.get("0"));

  console.log("===== FINAL STATUS =====");
  console.log("All tests completed.");
}

main();
