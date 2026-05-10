type TrieNode = {
  children: Map<string, TrieNode>;
  end: boolean;
  value: string | null;
};

const newNode = (): TrieNode => ({ children: new Map(), end: false, value: null });

export class Trie {
  private readonly root = newNode();

  insert(word: string, value = word): void {
    let node = this.root;
    for (const char of word) {
      let child = node.children.get(char);
      if (!child) {
        child = newNode();
        node.children.set(char, child);
      }
      node = child;
    }
    node.end = true;
    node.value = value;
  }

  search(word: string): string | null {
    const node = this.findNode(word);
    return node?.end ? node.value : null;
  }

  startsWith(prefix: string): boolean {
    return this.findNode(prefix) !== null;
  }

  delete(word: string): boolean {
    return this.deleteFrom(this.root, word, 0);
  }

  wordsWithPrefix(prefix: string): string[] {
    const start = this.findNode(prefix);
    if (!start) return [];
    const result: string[] = [];
    this.collect(start, prefix, result);
    return result;
  }

  private findNode(text: string): TrieNode | null {
    let node = this.root;
    for (const char of text) {
      const child = node.children.get(char);
      if (!child) return null;
      node = child;
    }
    return node;
  }

  private deleteFrom(node: TrieNode, word: string, index: number): boolean {
    if (index === word.length) {
      if (!node.end) return false;
      node.end = false;
      node.value = null;
      return true;
    }
    const char = word[index];
    const child = node.children.get(char);
    if (!child || !this.deleteFrom(child, word, index + 1)) return false;
    if (!child.end && child.children.size === 0) node.children.delete(char);
    return true;
  }

  private collect(node: TrieNode, word: string, result: string[]): void {
    if (node.end) result.push(word);
    for (const [char, child] of node.children) this.collect(child, word + char, result);
  }
}

function main(): void {
  const trie = new Trie();
  trie.insert("system");
  trie.insert("syscall");
  console.log(trie.search("system"), trie.wordsWithPrefix("sys"));
}

if (require.main === module) main();
