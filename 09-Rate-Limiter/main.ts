export class TokenBucketRateLimiter {
  private tokens: number;
  private lastRefillMs: number;

  constructor(private readonly capacity: number, private readonly refillPerSecond: number) {
    if (capacity <= 0 || refillPerSecond <= 0) throw new Error("capacity and refill rate must be positive");
    this.tokens = capacity;
    this.lastRefillMs = Date.now();
  }

  allow(cost = 1, nowMs = Date.now()): boolean {
    this.refill(nowMs);
    if (cost <= 0) return true;
    if (this.tokens < cost) return false;
    this.tokens -= cost;
    return true;
  }

  remaining(nowMs = Date.now()): number {
    this.refill(nowMs);
    return this.tokens;
  }

  private refill(nowMs: number): void {
    const elapsedSeconds = Math.max(0, nowMs - this.lastRefillMs) / 1000;
    this.tokens = Math.min(this.capacity, this.tokens + elapsedSeconds * this.refillPerSecond);
    this.lastRefillMs = nowMs;
  }
}

function main(): void {
  const limiter = new TokenBucketRateLimiter(2, 1);
  console.log(limiter.allow(), limiter.allow(), limiter.allow());
}

if (require.main === module) main();
