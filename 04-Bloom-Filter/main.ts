export class BloomFilter {
  private readonly bits: Uint8Array;
  private readonly bitSize: number;
  private readonly hashCount: number;

  constructor(expectedItems: number, falsePositiveRate = 0.01) {
    if (expectedItems <= 0) throw new Error("expectedItems must be positive");
    if (falsePositiveRate <= 0 || falsePositiveRate >= 1) {
      throw new Error("falsePositiveRate must be between 0 and 1");
    }

    const ln2 = Math.log(2);
    this.bitSize = Math.ceil((-expectedItems * Math.log(falsePositiveRate)) / (ln2 * ln2));
    this.hashCount = Math.max(1, Math.round((this.bitSize / expectedItems) * ln2));
    this.bits = new Uint8Array(Math.ceil(this.bitSize / 8));
  }

  add(value: string): void {
    for (const index of this.indexes(value)) this.setBit(index);
  }

  has(value: string): boolean {
    for (const index of this.indexes(value)) {
      if (!this.getBit(index)) return false;
    }
    return true;
  }

  estimatedFalsePositiveRate(insertedItems: number): number {
    return Math.pow(1 - Math.exp((-this.hashCount * insertedItems) / this.bitSize), this.hashCount);
  }

  private indexes(value: string): number[] {
    const h1 = this.fnv1a(value, 0x811c9dc5);
    const h2 = this.fnv1a(value, 0x01000193) || 1;
    const result: number[] = [];
    for (let i = 0; i < this.hashCount; i++) result.push((h1 + i * h2) % this.bitSize);
    return result;
  }

  private fnv1a(value: string, seed: number): number {
    let hash = seed >>> 0;
    for (let i = 0; i < value.length; i++) {
      hash ^= value.charCodeAt(i);
      hash = Math.imul(hash, 16777619) >>> 0;
    }
    return hash;
  }

  private setBit(index: number): void {
    this.bits[index >> 3] |= 1 << (index & 7);
  }

  private getBit(index: number): boolean {
    return (this.bits[index >> 3] & (1 << (index & 7))) !== 0;
  }
}

function main(): void {
  const filter = new BloomFilter(1000, 0.01);
  filter.add("alice");
  filter.add("bob");
  console.log(filter.has("alice"), filter.has("carol"));
}

if (require.main === module) main();
