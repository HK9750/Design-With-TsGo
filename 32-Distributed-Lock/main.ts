type Lease = { owner: string; token: number; expiresAt: number };

export class DistributedLockManager {
  private readonly locks = new Map<string, Lease>();
  private nextToken = 1;

  acquire(resource: string, owner: string, ttlMs: number, nowMs = Date.now()): Lease | null {
    const current = this.locks.get(resource);
    if (current && current.expiresAt > nowMs) return null;
    const lease = { owner, token: this.nextToken++, expiresAt: nowMs + ttlMs };
    this.locks.set(resource, lease);
    return lease;
  }

  release(resource: string, owner: string, token: number): boolean {
    const current = this.locks.get(resource);
    if (!current || current.owner !== owner || current.token !== token) return false;
    this.locks.delete(resource);
    return true;
  }

  validate(resource: string, token: number, nowMs = Date.now()): boolean {
    const lease = this.locks.get(resource);
    return !!lease && lease.token === token && lease.expiresAt > nowMs;
  }
}

function main(): void {
  const manager = new DistributedLockManager();
  const lease = manager.acquire("order:1", "worker-a", 1000);
  console.log(lease?.token, lease ? manager.validate("order:1", lease.token) : false);
}

if (require.main === module) main();
