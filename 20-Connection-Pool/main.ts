export class ConnectionPool<T> {
  private readonly idle: T[] = [];
  private active = 0;

  constructor(private readonly max: number, private readonly factory: () => T, private readonly validate: (conn: T) => boolean = () => true) {}

  acquire(): T {
    while (this.idle.length > 0) {
      const conn = this.idle.pop()!;
      if (this.validate(conn)) {
        this.active++;
        return conn;
      }
    }
    if (this.active >= this.max) throw new Error("connection pool exhausted");
    this.active++;
    return this.factory();
  }

  release(conn: T): void {
    this.active--;
    if (this.validate(conn)) this.idle.push(conn);
  }
}

function main(): void {
  const pool = new ConnectionPool(2, () => ({ id: Math.random() }));
  const conn = pool.acquire();
  pool.release(conn);
  console.log("connection returned");
}

if (require.main === module) main();
