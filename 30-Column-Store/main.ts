export type ColumnValue = string | number | boolean | null;
export type Row = Record<string, ColumnValue>;

export class ColumnStore {
  private readonly columns = new Map<string, ColumnValue[]>();
  private rows = 0;

  insert(row: Row): void {
    for (const column of this.columns.keys()) this.columns.get(column)!.push(row[column] ?? null);
    for (const [column, value] of Object.entries(row)) {
      if (!this.columns.has(column)) this.columns.set(column, new Array(this.rows).fill(null));
      this.columns.get(column)!.push(value);
    }
    this.rows++;
  }

  project(columns: string[]): Row[] {
    const result: Row[] = [];
    for (let i = 0; i < this.rows; i++) {
      const row: Row = {};
      for (const column of columns) row[column] = this.columns.get(column)?.[i] ?? null;
      result.push(row);
    }
    return result;
  }
}

function main(): void {
  const store = new ColumnStore();
  store.insert({ id: 1, country: "PK" });
  store.insert({ id: 2, country: "US" });
  console.log(store.project(["country"]));
}

if (require.main === module) main();
