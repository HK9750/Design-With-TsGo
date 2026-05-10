export type SagaStep = { name: string; action: () => Promise<void>; compensate: () => Promise<void> };

export class SagaCoordinator {
  constructor(private readonly steps: SagaStep[]) {}

  async execute(): Promise<boolean> {
    const completed: SagaStep[] = [];
    try {
      for (const step of this.steps) {
        await step.action();
        completed.push(step);
      }
      return true;
    } catch {
      for (const step of completed.reverse()) await step.compensate();
      return false;
    }
  }
}

async function main(): Promise<void> {
  const events: string[] = [];
  const saga = new SagaCoordinator([
    { name: "order", action: async () => { events.push("order"); }, compensate: async () => { events.push("undo-order"); } },
  ]);
  console.log(await saga.execute(), events);
}

if (require.main === module) void main();
