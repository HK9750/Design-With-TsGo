const FNV_PRIMES = {
  32: 16_777_619n,
  64: 1_099_511_628_211n,
};

const FNV_OFFSETS = {
  32: 2_166_136_261n,
  64: 14_695_981_039_346_656_037n,
};

export type FNV_SIZE = keyof typeof FNV_PRIMES;

export const fnv1a = (buffer: Uint8Array, size: FNV_SIZE): bigint => {
  const prime = FNV_PRIMES[size];
  let hash = FNV_OFFSETS[size];

  for (let i = 0; i < buffer.length; i++) {
    hash ^= BigInt(buffer[i]);
    hash = BigInt.asUintN(size, hash * prime);
  }

  return hash;
};

export const fnv1a_on_string = (key: string, size: FNV_SIZE): bigint => {
  const bytes = new TextEncoder().encode(key);
  return fnv1a(bytes, size);
};
