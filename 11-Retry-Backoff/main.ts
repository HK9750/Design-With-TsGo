export type RetryPolicy = {
  maxAttempts: number;
  initialDelayMs: number;
  maxDelayMs: number;
  multiplier: number;
  jitter?: boolean;
};

export async function retry<T>(operation: (attempt: number) => Promise<T>, policy: RetryPolicy): Promise<T> {
  let lastError: unknown;
  for (let attempt = 1; attempt <= policy.maxAttempts; attempt++) {
    try {
      return await operation(attempt);
    } catch (error) {
      lastError = error;
      if (attempt === policy.maxAttempts) break;
      await sleep(delayForAttempt(attempt, policy));
    }
  }
  throw lastError;
}

export function delayForAttempt(attempt: number, policy: RetryPolicy): number {
  const exponential = policy.initialDelayMs * Math.pow(policy.multiplier, attempt - 1);
  const capped = Math.min(policy.maxDelayMs, exponential);
  return policy.jitter ? Math.floor(Math.random() * capped) : capped;
}

const sleep = (ms: number): Promise<void> => new Promise((resolve) => setTimeout(resolve, ms));

async function main(): Promise<void> {
  const value = await retry(async (attempt) => {
    if (attempt < 2) throw new Error("transient");
    return "ok";
  }, { maxAttempts: 3, initialDelayMs: 10, maxDelayMs: 100, multiplier: 2, jitter: false });
  console.log(value);
}

if (require.main === module) void main();
