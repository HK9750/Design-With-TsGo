export class InvertedIndex {
  private readonly postings = new Map<string, Map<string, number>>();

  addDocument(id: string, text: string): void {
    const counts = new Map<string, number>();
    for (const term of tokenize(text)) counts.set(term, (counts.get(term) ?? 0) + 1);
    for (const [term, count] of counts) {
      let posting = this.postings.get(term);
      if (!posting) this.postings.set(term, posting = new Map());
      posting.set(id, count);
    }
  }

  searchAll(query: string): string[] {
    const terms = tokenize(query);
    if (terms.length === 0) return [];
    let result = new Set(this.postings.get(terms[0])?.keys() ?? []);
    for (const term of terms.slice(1)) result = intersect(result, new Set(this.postings.get(term)?.keys() ?? []));
    return [...result].sort();
  }
}

function tokenize(text: string): string[] {
  return text.toLowerCase().match(/[a-z0-9]+/g) ?? [];
}

function intersect(a: Set<string>, b: Set<string>): Set<string> {
  return new Set([...a].filter((value) => b.has(value)));
}

function main(): void {
  const index = new InvertedIndex();
  index.addDocument("1", "distributed systems design");
  index.addDocument("2", "systems programming");
  console.log(index.searchAll("systems design"));
}

if (require.main === module) main();
