type SkipNode<V> = {
  key: number;
  value: V | null;
  forward: Array<SkipNode<V> | null>;
};

export class SkipList<V> {
  private readonly head: SkipNode<V>;
  private level = 0;

  constructor(private readonly maxLevel = 16, private readonly probability = 0.5) {
    this.head = { key: Number.NEGATIVE_INFINITY, value: null, forward: new Array(maxLevel + 1).fill(null) };
  }

  search(key: number): V | null {
    let node = this.head;
    for (let i = this.level; i >= 0; i--) {
      while (node.forward[i] && node.forward[i]!.key < key) node = node.forward[i]!;
    }
    node = node.forward[0] ?? this.head;
    return node.key === key ? node.value : null;
  }

  insert(key: number, value: V): void {
    const update = new Array<SkipNode<V>>(this.maxLevel + 1);
    let node = this.head;
    for (let i = this.level; i >= 0; i--) {
      while (node.forward[i] && node.forward[i]!.key < key) node = node.forward[i]!;
      update[i] = node;
    }

    const existing = node.forward[0];
    if (existing?.key === key) {
      existing.value = value;
      return;
    }

    const newLevel = this.randomLevel();
    if (newLevel > this.level) {
      for (let i = this.level + 1; i <= newLevel; i++) update[i] = this.head;
      this.level = newLevel;
    }

    const created: SkipNode<V> = { key, value, forward: new Array(newLevel + 1).fill(null) };
    for (let i = 0; i <= newLevel; i++) {
      created.forward[i] = update[i].forward[i];
      update[i].forward[i] = created;
    }
  }

  delete(key: number): boolean {
    const update = new Array<SkipNode<V>>(this.maxLevel + 1);
    let node = this.head;
    for (let i = this.level; i >= 0; i--) {
      while (node.forward[i] && node.forward[i]!.key < key) node = node.forward[i]!;
      update[i] = node;
    }

    const target = node.forward[0];
    if (!target || target.key !== key) return false;

    for (let i = 0; i <= this.level && update[i].forward[i] === target; i++) {
      update[i].forward[i] = target.forward[i] ?? null;
    }
    while (this.level > 0 && !this.head.forward[this.level]) this.level--;
    return true;
  }

  entries(): Array<[number, V]> {
    const result: Array<[number, V]> = [];
    let node = this.head.forward[0];
    while (node) {
      result.push([node.key, node.value as V]);
      node = node.forward[0];
    }
    return result;
  }

  private randomLevel(): number {
    let level = 0;
    while (level < this.maxLevel && Math.random() < this.probability) level++;
    return level;
  }
}

function main(): void {
  const list = new SkipList<string>();
  list.insert(3, "three");
  list.insert(1, "one");
  list.insert(2, "two");
  console.log(list.search(2), list.entries());
}

if (require.main === module) main();
