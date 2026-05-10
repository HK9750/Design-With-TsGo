export interface Participant {
  prepare(transactionId: string): boolean;
  commit(transactionId: string): void;
  abort(transactionId: string): void;
}

export class TwoPhaseCommitCoordinator {
  constructor(private readonly participants: Participant[]) {}

  execute(transactionId: string): "committed" | "aborted" {
    const prepared: Participant[] = [];
    for (const participant of this.participants) {
      if (!participant.prepare(transactionId)) {
        for (const p of prepared) p.abort(transactionId);
        return "aborted";
      }
      prepared.push(participant);
    }
    for (const participant of prepared) participant.commit(transactionId);
    return "committed";
  }
}

function main(): void {
  const participant: Participant = { prepare: () => true, commit: () => {}, abort: () => {} };
  console.log(new TwoPhaseCommitCoordinator([participant]).execute("tx1"));
}

if (require.main === module) main();
