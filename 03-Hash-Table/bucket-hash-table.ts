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
//

type Entry<K, V> = {
  key: K;
  value: V;
  next?: Entry<K, V>;
};

export class HashMap<K extends string, V> {
  private buckets: Array<Entry<K, V> | undefined>;
  private size: number;
  private capacity: number;
  private minimumThreshold = 0.25;
  private maximumThreshold = 0.75;

  constructor(capacity: number = 16) {
    this.size = 0;
    this.capacity = capacity;
    this.buckets = new Array(this.capacity);
  }

  private getThreshold() {
    return this.size / this.capacity;
  }

  get(key: K): V | undefined {
    return undefined;
  }

  set(key: K, value: V): void {}

  delete(key: K): Boolean {
    return true;
  }

  private resize(capacity: number) {}
}
