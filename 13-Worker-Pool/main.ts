export class WorkerPool<I, O> {
  constructor(private readonly workers: number, private readonly handler: (job: I) => Promise<O> | O) {
    if (workers <= 0) throw new Error("workers must be positive");
  }

  async run(jobs: I[]): Promise<O[]> {
    const results = new Array<O>(jobs.length);
    let next = 0;

    const worker = async (): Promise<void> => {
      for (;;) {
        const index = next++;
        if (index >= jobs.length) return;
        results[index] = await this.handler(jobs[index]);
      }
    };

    await Promise.all(Array.from({ length: Math.min(this.workers, jobs.length) }, worker));
    return results;
  }
}

async function main(): Promise<void> {
  const pool = new WorkerPool<number, number>(3, async (n) => n * n);
  console.log(await pool.run([1, 2, 3, 4]));
}

if (require.main === module) void main();
