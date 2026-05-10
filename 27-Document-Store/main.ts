export type Document = Record<string, string | number | boolean>;

export class DocumentStore {
  private readonly docs = new Map<string, Document>();
  private readonly indexes = new Map<string, Map<string, Set<string>>>();

  createIndex(field: string): void {
    const index = new Map<string, Set<string>>();
    for (const [id, doc] of this.docs) addToIndex(index, String(doc[field]), id);
    this.indexes.set(field, index);
  }

  upsert(id: string, doc: Document): void {
    this.docs.set(id, doc);
    for (const [field, index] of this.indexes) addToIndex(index, String(doc[field]), id);
  }

  findByField(field: string, value: string | number | boolean): Document[] {
    const index = this.indexes.get(field);
    const ids = index?.get(String(value));
    if (ids) return [...ids].map((id) => this.docs.get(id)!).filter(Boolean);
    return [...this.docs.values()].filter((doc) => doc[field] === value);
  }
}

function addToIndex(index: Map<string, Set<string>>, value: string, id: string): void {
  let ids = index.get(value);
  if (!ids) index.set(value, ids = new Set());
  ids.add(id);
}

function main(): void {
  const store = new DocumentStore();
  store.upsert("1", { type: "user", name: "Ada" });
  store.createIndex("type");
  console.log(store.findByField("type", "user"));
}

if (require.main === module) main();
