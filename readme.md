# Design With TypeScript & Go 🚀

This repository is a daily record of my journey learning and implementing system design concepts, data structures, and design patterns. Each topic is implemented in both **TypeScript** and **Go** to compare and contrast the approaches in different paradigms.

The goal is backend-engineering depth: understand what problem each pattern solves, where it appears in production, and how to evolve the learning implementation into a production-ready component.

Start with the production guide: [Backend Engineering Production Guide](./docs/backend-engineering-production-guide.md)

## 📚 Topics Covered

- [x] 01 - [LRU Cache](./01-LRU-Cache)
- [x] 02 - [LFU Cache](./02-LFU-Cache)
- [x] 03 - [Hash Table](./03-Hash-Table)
- [x] 04 - [Bloom Filter](./04-Bloom-Filter)
- [x] 05 - [Skip List](./05-Skip-List)
- [x] 06 - [Min Heap / Max Heap](./06-Min-Heap-Max-Heap)
- [x] 07 - [Trie](./07-Trie)
- [x] 08 - [Consistent Hashing](./08-Consistent-Hashing)
- [x] 09 - [Rate Limiter](./09-Rate-Limiter)
- [x] 10 - [Circuit Breaker](./10-Circuit-Breaker)
- [x] 11 - [Retry Backoff](./11-Retry-Backoff)
- [x] 12 - [ID Generator](./12-ID-Generator)
- [x] 13 - [Worker Pool](./13-Worker-Pool)
- [x] 14 - [Promise](./14-Promise)
- [x] 15 - [Semaphore](./15-Semaphore)
- [x] 16 - [Read Write Lock](./16-Read-Write-Lock)
- [x] 17 - [Channel Pool](./17-Channel-Pool)
- [x] 18 - [Fan Out / Fan In](./18-Fan-Out-Fan-In)
- [x] 19 - [Producer Consumer](./19-Producer-Consumer)
- [x] 20 - [Connection Pool](./20-Connection-Pool)
- [x] 21 - [B-Tree](./21-B-Tree)
- [x] 22 - [LSM Tree](./22-LSM-Tree)
- [x] 23 - [Write-Ahead Log](./23-Write-Ahead-Log)
- [x] 24 - [Inverted Index](./24-Inverted-Index)
- [x] 25 - [Time-Series Store](./25-Time-Series-Store)
- [x] 26 - [Key-Value Store](./26-Key-Value-Store)
- [x] 27 - [Document Store](./27-Document-Store)
- [x] 28 - [Graph Database](./28-Graph-Database)
- [x] 29 - [Geospatial Index](./29-Geospatial-Index)
- [x] 30 - [Column Store](./30-Column-Store)
- [x] 31 - [Leader Election](./31-Leader-Election)
- [x] 32 - [Distributed Lock](./32-Distributed-Lock)
- [x] 33 - [Gossip Protocol](./33-Gossip-Protocol)
- [x] 34 - [Vector Clocks](./34-Vector-Clocks)
- [x] 35 - [CRDT](./35-CRDT)
- [x] 36 - [Distributed Transaction](./36-Distributed-Transaction)
- [x] 37 - [Two-Phase Commit](./37-Two-Phase-Commit)
- [x] 38 - [Replication](./38-Replication)
- [x] 39 - [Sharding](./39-Sharding)
- [x] 40 - [Consensus](./40-Consensus)

## 🛠 How to Run

### Go
To run the Go implementation:
```bash
go run <topic-folder>/main.go
```
Example:
```bash
go run 09-Rate-Limiter/main.go
```

### TypeScript
Build and run the TypeScript implementation:
```bash
npm run build
node dist/<topic-folder>/main.js
```
Example:
```bash
npm run build
node dist/09-Rate-Limiter/main.js
```

### Verify Everything
```bash
npx tsc --noEmit
GO111MODULE=off go test ./...
```

---
Happy Coding!
