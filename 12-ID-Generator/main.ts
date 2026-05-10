export class SnowflakeGenerator {
  private sequence = 0n;
  private lastTimestamp = -1n;

  constructor(private readonly machineId: bigint, private readonly epochMs = 1704067200000n) {
    if (machineId < 0n || machineId > 1023n) throw new Error("machineId must fit in 10 bits");
  }

  nextId(nowMs = BigInt(Date.now())): bigint {
    let timestamp = nowMs - this.epochMs;
    if (timestamp < this.lastTimestamp) throw new Error("clock moved backwards");

    if (timestamp === this.lastTimestamp) {
      this.sequence = (this.sequence + 1n) & 4095n;
      if (this.sequence === 0n) {
        do timestamp = BigInt(Date.now()) - this.epochMs;
        while (timestamp <= this.lastTimestamp);
      }
    } else {
      this.sequence = 0n;
    }

    this.lastTimestamp = timestamp;
    return (timestamp << 22n) | (this.machineId << 12n) | this.sequence;
  }
}

function main(): void {
  const generator = new SnowflakeGenerator(7n);
  console.log(generator.nextId().toString());
}

if (require.main === module) main();
