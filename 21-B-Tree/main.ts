class BTreeNode {
  keys: number[] = [];
  children: BTreeNode[] = [];
  leaf = true;
}

export class BTree {
  private root = new BTreeNode();

  constructor(private readonly minDegree = 2) {
    if (minDegree < 2) throw new Error("minDegree must be at least 2");
  }

  search(key: number, node = this.root): boolean {
    let i = 0;
    while (i < node.keys.length && key > node.keys[i]) i++;
    if (i < node.keys.length && key === node.keys[i]) return true;
    if (node.leaf) return false;
    return this.search(key, node.children[i]);
  }

  insert(key: number): void {
    const root = this.root;
    if (root.keys.length === 2 * this.minDegree - 1) {
      const nextRoot = new BTreeNode();
      nextRoot.leaf = false;
      nextRoot.children[0] = root;
      this.splitChild(nextRoot, 0);
      this.root = nextRoot;
    }
    this.insertNonFull(this.root, key);
  }

  private insertNonFull(node: BTreeNode, key: number): void {
    let i = node.keys.length - 1;
    if (node.leaf) {
      while (i >= 0 && key < node.keys[i]) i--;
      if (node.keys[i] === key) return;
      node.keys.splice(i + 1, 0, key);
      return;
    }
    while (i >= 0 && key < node.keys[i]) i--;
    i++;
    if (node.children[i].keys.length === 2 * this.minDegree - 1) {
      this.splitChild(node, i);
      if (key > node.keys[i]) i++;
    }
    this.insertNonFull(node.children[i], key);
  }

  private splitChild(parent: BTreeNode, index: number): void {
    const full = parent.children[index];
    const sibling = new BTreeNode();
    sibling.leaf = full.leaf;
    const middle = full.keys[this.minDegree - 1];
    sibling.keys = full.keys.splice(this.minDegree);
    full.keys.splice(this.minDegree - 1);
    if (!full.leaf) sibling.children = full.children.splice(this.minDegree);
    parent.keys.splice(index, 0, middle);
    parent.children.splice(index + 1, 0, sibling);
  }
}

function main(): void {
  const tree = new BTree(2);
  [10, 20, 5, 6, 12, 30, 7, 17].forEach((n) => tree.insert(n));
  console.log(tree.search(12), tree.search(99));
}

if (require.main === module) main();
