export type CircuitState = "closed" | "open" | "half-open";

export class CircuitBreaker<T> {
  private state: CircuitState = "closed";
  private failures = 0;
  private openedAt = 0;

  constructor(private readonly failureThreshold: number, private readonly openTimeoutMs: number) {}

  getState(): CircuitState {
    if (this.state === "open" && Date.now() - this.openedAt >= this.openTimeoutMs) this.state = "half-open";
    return this.state;
  }

  async execute(operation: () => Promise<T>, fallback?: () => T | Promise<T>): Promise<T> {
    if (this.getState() === "open") {
      if (fallback) return fallback();
      throw new Error("circuit breaker is open");
    }

    try {
      const value = await operation();
      this.recordSuccess();
      return value;
    } catch (error) {
      this.recordFailure();
      if (fallback) return fallback();
      throw error;
    }
  }

  private recordSuccess(): void {
    this.failures = 0;
    this.state = "closed";
  }

  private recordFailure(): void {
    this.failures++;
    if (this.state === "half-open" || this.failures >= this.failureThreshold) {
      this.state = "open";
      this.openedAt = Date.now();
    }
  }
}

async function main(): Promise<void> {
  const breaker = new CircuitBreaker<string>(2, 1000);
  console.log(await breaker.execute(async () => "ok"));
}

if (require.main === module) void main();
