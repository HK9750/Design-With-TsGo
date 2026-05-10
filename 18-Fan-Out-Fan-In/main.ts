export async function fanOutFanIn<I, O>(items: I[], workers: number, handler: (item: I) => Promise<O> | O): Promise<O[]> {
  let index = 0;
  const results = new Array<O>(items.length);
  await Promise.all(Array.from({ length: Math.min(workers, items.length) }, async () => {
    for (;;) {
      const current = index++;
      if (current >= items.length) return;
      results[current] = await handler(items[current]);
    }
  }));
  return results;
}

async function main(): Promise<void> {
  console.log(await fanOutFanIn([1, 2, 3], 2, async (n) => n * 10));
}

if (require.main === module) void main();
