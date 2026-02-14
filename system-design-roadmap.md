# Backend System Design Roadmap

A comprehensive learning path for implementing 52 system design topics in TypeScript and Go.

---

## Overview

This roadmap is designed for backend engineers to build production-ready system design implementations from scratch. Each topic builds upon previous concepts, creating a solid foundation in distributed systems, storage engines, and real-world architectures.

**Prerequisites:**
- Solid understanding of Go and TypeScript
- Familiarity with data structures and algorithms
- Basic networking and concurrency concepts

---

## Phase 1: Core Data Structures (Fundamentals)

### 01-LRU-Cache
**Status:** ✅ Implemented  
**Complexity:** Entry-level

**Architecture Overview:**
- **Data Structures:** Hash map + Doubly linked list
- **Access Pattern:** O(1) get/put operations
- **Eviction Policy:** Least Recently Used
- **Memory Model:** Fixed capacity with automatic eviction

**Key Components:**
1. **Hash Map:** Maps keys to list nodes for O(1) lookup
2. **Doubly Linked List:** Maintains access order (head = most recent, tail = least recent)
3. **Sentinel Nodes:** Dummy head/tail for simplified edge case handling

**Design Patterns:**
- Object Pool Pattern (node reuse)
- Proxy Pattern (cache as transparent layer)

**Learning Outcomes:**
- Combining data structures for optimized operations
- Memory management with fixed capacity
- Pointer manipulation (Go) / Reference handling (TS)

---

### 02-LFU-Cache
**Status:** ✅ Implemented  
**Complexity:** Entry-level

**Architecture Overview:**
- **Data Structures:** Hash map + Multiple doubly linked lists + Min-heap or frequency map
- **Access Pattern:** O(1) operations with frequency tracking
- **Eviction Policy:** Least Frequently Used
- **Memory Model:** Hierarchical structure per frequency level

**Key Components:**
1. **Main Hash Map:** Key → Node mapping
2. **Frequency Map:** Frequency → Linked List of nodes
3. **Min Frequency Tracker:** Tracks lowest frequency for eviction
4. **Node Structure:** Extends LRU node with frequency counter

**Design Patterns:**
- Composite Pattern (frequency lists)
- Strategy Pattern (eviction policies)

**Learning Outcomes:**
- Multi-level data structure organization
- Frequency-based algorithms
- Complex pointer chains management

---

### 03-Hash-Table
**Complexity:** Entry-level

**Architecture Overview:**
- **Data Structures:** Array of buckets + Linked lists or open addressing
- **Access Pattern:** O(1) average, O(n) worst case
- **Collision Handling:** Chaining or Open Addressing (Linear/Quadratic probing)
- **Resizing Strategy:** Dynamic resizing with rehashing

**Key Components:**
1. **Buckets Array:** Fixed-size array holding entries
2. **Hash Function:** Deterministic key → index mapping (djb2, FNV-1a)
3. **Collision Resolution:** 
   - Chaining: Linked list per bucket
   - Open Addressing: Probe sequence until empty slot
4. **Load Factor:** Threshold (0.75) triggering resize
5. **Rehashing:** Expands array and redistributes entries

**Design Patterns:**
- Iterator Pattern (key-value traversal)
- Flyweight Pattern (entry reuse)

**Algorithms:**
- Universal hashing
- Robin Hood hashing (optional optimization)

**Learning Outcomes:**
- Hash function design principles
- Collision handling strategies
- Amortized complexity analysis
- Memory reallocation patterns

---

### 04-Bloom-Filter
**Complexity:** Entry-level

**Architecture Overview:**
- **Data Structures:** Bit array + Multiple hash functions
- **Access Pattern:** O(k) where k = number of hash functions
- **Probabilistic Model:** False positives possible, false negatives impossible
- **Space Efficiency:** ~10x less memory than hash sets

**Key Components:**
1. **Bit Array:** m bits initialized to 0
2. **Hash Functions:** k independent hash functions
3. **Insert Operation:** Set bits at all k positions
4. **Query Operation:** Check if all k bits are set
5. **Parameters:** 
   - n = expected elements
   - p = false positive rate
   - m = optimal bit array size = -n*ln(p)/(ln(2)²)
   - k = optimal hash functions = m/n * ln(2)

**Design Patterns:**
- Probabilistic Data Structure Pattern

**Algorithms:**
- MurmurHash, FNV family
- Double hashing technique for generating k hashes

**Variants to Consider:**
- Counting Bloom Filter (allows deletion)
- Scalable Bloom Filter (dynamic sizing)
- Cuckoo Filter (better cache locality)

**Learning Outcomes:**
- Probabilistic data structures trade-offs
- Hash function selection
- Space-time complexity trade-offs
- False positive rate calculation

---

### 05-Skip-List
**Complexity:** Entry-level

**Architecture Overview:**
- **Data Structures:** Multi-level linked lists with probabilistic promotion
- **Access Pattern:** O(log n) average, O(n) worst case
- **Height:** log(n) levels on average
- **Probabilistic Balancing:** Coin flip determines level promotion

**Key Components:**
1. **Nodes:** Value + array of forward pointers (one per level)
2. **Levels:** 0 to max level, each level skips more elements
3. **Head Node:** Sentinel with max level pointers
4. **Promotion Strategy:** Coin flip (p = 0.5) or geometric distribution
5. **Max Level:** log₁/ₚ(n) to keep operations logarithmic

**Operations:**
- **Search:** Start at highest level, move forward or drop down
- **Insert:** Find position at each level, insert and promote probabilistically
- **Delete:** Remove from all levels containing the node

**Design Patterns:**
- Composite Pattern (multi-level structure)
- Lazy Propagation (optional optimization)

**Algorithms:**
- Randomized level generation
- Lock-free skip list algorithms (advanced)

**Learning Outcomes:**
- Probabilistic balancing without complex rebalancing
- Multi-level pointer navigation
- Expected value analysis
- Alternative to balanced trees

---

### 06-Min-Heap-Max-Heap
**Complexity:** Entry-level

**Architecture Overview:**
- **Data Structures:** Complete binary tree stored in array
- **Access Pattern:** O(log n) insert/extract, O(1) peek
- **Heap Property:** Parent ≤ children (min-heap) or parent ≥ children (max-heap)
- **Complete Tree:** All levels filled except possibly last, filled left-to-right

**Key Components:**
1. **Array Storage:** Index i has children at 2i+1 and 2i+2
2. **Heapify:** Maintain heap property bottom-up or top-down
3. **Sift Up:** Move element up after insertion
4. **Sift Down:** Move element down after extraction
5. **Priority Queue Interface:** Insert, Extract-Min/Max, Peek

**Operations:**
- **Insert:** Add at end, sift up
- **Extract-Min/Max:** Swap with last, remove, sift down root
- **Build-Heap:** O(n) heapify all elements
- **Heapify:** O(log n) restore heap property

**Design Patterns:**
- Priority Queue Pattern
- Strategy Pattern (min vs max comparator)

**Variants:**
- Binomial Heap
- Fibonacci Heap
- Pairing Heap

**Learning Outcomes:**
- Complete binary tree properties
- Array-based tree representation
- Heap sort algorithm
- Priority queue implementations

---

### 07-Trie (Prefix Tree)
**Complexity:** Entry-level

**Architecture Overview:**
- **Data Structures:** N-ary tree (typically 26 for letters, 256 for bytes)
- **Access Pattern:** O(m) where m = key length
- **Prefix Matching:** Natural support for prefix-based operations
- **Space Trade-off:** High memory for fast lookups

**Key Components:**
1. **TrieNode:** Children map + isEndOfWord flag + optional value
2. **Root Node:** Empty node, entry point
3. **Edge Labels:** Characters/bytes labeling edges to children
4. **Terminal Nodes:** Marked with isEndOfWord = true

**Operations:**
- **Insert:** Create path for each character, mark terminal
- **Search:** Traverse path, check terminal flag
- **StartsWith:** Traverse without requiring terminal node
- **Delete:** Remove nodes only if not part of other words

**Optimizations:**
- **Compressed Trie (Radix Tree):** Merge single-child nodes
- **Ternary Search Tree:** Space-efficient alternative

**Design Patterns:**
- Tree Pattern
- Flyweight Pattern (shared subtrees)

**Applications:**
- Autocomplete systems
- Spell checking
- IP routing tables
- Dictionary implementations

**Learning Outcomes:**
- Prefix-based data structures
- String processing algorithms
- Space-time trade-offs in trees

---

### 08-Consistent-Hashing
**Complexity:** Intermediate

**Architecture Overview:**
- **Data Structures:** Hash ring (circular space) + Sorted map of virtual nodes
- **Access Pattern:** O(log n) for node lookup
- **Rebalancing:** Minimal key remapping on node add/remove
- **Virtual Nodes:** Multiple replicas per physical node for better distribution

**Key Components:**
1. **Hash Ring:** Circular space [0, 2³²-1] or [0, 2¹⁶⁰-1]
2. **Node Positions:** Hash(node_id) mod ring_size
3. **Virtual Nodes:** Each physical node gets v virtual replicas
4. **Key Mapping:** Hash(key) → clockwise to first node
5. **Sorted Map:** Binary search for O(log n) lookups

**Operations:**
- **Add Node:** Add v virtual nodes, redistribute affected keys
- **Remove Node:** Remove v virtual nodes, redistribute keys
- **Get Node:** Hash(key), binary search for successor node

**Algorithms:**
- Ring traversal with virtual node replication
- Rendezvous hashing (alternative)
- Jump consistent hash (alternative for monotonicity)

**Design Patterns:**
- Strategy Pattern (hash function selection)
- Proxy Pattern (transparent routing)

**Real-World Applications:**
- DynamoDB partition keys
- Memcached sharding
- CDN request routing
- Distributed caching

**Learning Outcomes:**
- Distributed data partitioning
- Hash ring mechanics
- Virtual node advantages
- Minimal rebalancing strategies

---

### 09-Rate-Limiter
**Complexity:** Intermediate

**Architecture Overview:**
- **Algorithms:** Token Bucket, Leaky Bucket, Fixed/Sliding Window
- **State Management:** In-memory or distributed (Redis)
- **Precision:** Millisecond-level tracking
- **Burst Handling:** Configurable burst capacity

**Key Components:**

**Token Bucket:**
1. **Bucket:** Fixed capacity storing tokens
2. **Tokens:** Added at constant rate (refill rate)
3. **Request Handling:** Consume tokens, reject if insufficient
4. **Refill Strategy:** Add tokens every interval or on-demand

**Leaky Bucket:**
1. **Queue:** Fixed-size FIFO queue
2. **Leak Rate:** Constant outflow rate
3. **Request Handling:** Queue request, drop if full
4. **Processing:** Process at constant rate

**Sliding Window:**
1. **Time Windows:** Track request timestamps
2. **Counter:** Count requests in current window
3. **Cleanup:** Remove expired timestamps
4. **Decision:** Reject if count > limit

**Design Patterns:**
- Strategy Pattern (algorithm selection)
- Decorator Pattern (wrap any function)
- Sliding Window Pattern

**Distributed Considerations:**
- Redis with Lua scripts for atomicity
- Token bucket with Redis cell module
- Sliding window log in Redis sorted sets

**Learning Outcomes:**
- Rate limiting algorithms comparison
- Time-based state management
- Distributed state synchronization
- Algorithmic fairness and burst handling

---

### 10-Circuit-Breaker
**Complexity:** Intermediate

**Architecture Overview:**
- **States:** CLOSED, OPEN, HALF-OPEN
- **Transition Logic:** Failure threshold and timeout-based recovery
- **Monitoring:** Request count, failure rate, latency tracking
- **Isolation:** Prevents cascade failures

**Key Components:**
1. **State Machine:** Three states with transition rules
   - **CLOSED:** Normal operation, track failures
   - **OPEN:** Block requests, return fallback
   - **HALF-OPEN:** Test with limited requests
2. **Failure Metrics:** Rolling window counters
3. **Threshold Config:** Failure rate, request volume, timeout
4. **Fallback Strategy:** Default response or error
5. **Half-Open Probe:** Limited requests to test recovery

**Operations:**
- **Call:** Execute wrapped function
- **Record Success:** Update metrics, potentially close circuit
- **Record Failure:** Update metrics, potentially open circuit
- **State Check:** Determine if request should proceed

**Design Patterns:**
- State Pattern (state transitions)
- Proxy Pattern (transparent wrapping)
- Decorator Pattern (function wrapper)
- Observer Pattern (metrics publishing)

**Advanced Features:**
- Adaptive timeouts based on latency percentiles
- Different thresholds for different error types
- Bulkhead pattern integration (resource isolation)

**Learning Outcomes:**
- Fault tolerance patterns
- State machine implementation
- Metrics collection and thresholds
- Resilient architecture design

---

### 11-Retry-Backoff
**Complexity:** Entry-level

**Architecture Overview:**
- **Strategies:** Fixed, Linear, Exponential, Exponential with Jitter
- **State Management:** Attempt counter, last error, backoff deadline
- **Cancellations:** Context-aware cancellation support
- **Idempotency:** Safe retry semantics

**Key Components:**
1. **Retry Policy:**
   - Max attempts
   - Initial delay
   - Max delay
   - Multiplier (for exponential)
   - Jitter strategy
2. **Backoff Algorithms:**
   - **Fixed:** Constant delay
   - **Linear:** delay = initial + (attempt * increment)
   - **Exponential:** delay = initial * (multiplier^attempt)
   - **Decorrelated Jitter:** Randomized to prevent thundering herd
3. **Retryable Errors:** Whitelist of transient errors
4. **Context Integration:** Respect cancellation and deadlines

**Design Patterns:**
- Strategy Pattern (backoff algorithms)
- Decorator Pattern (function retry wrapper)
- Template Method Pattern (retry loop structure)

**Jitter Strategies:**
- **Full Jitter:** sleep = random(0, calculated_delay)
- **Equal Jitter:** sleep = (calculated_delay / 2) + random(0, calculated_delay / 2)
- **Decorrelated Jitter:** sleep = random(min_delay, previous_delay * 3)

**Learning Outcomes:**
- Retry semantics and safety
- Backoff algorithm selection
- Jitter importance in distributed systems
- Context-aware operations

---

### 12-ID-Generator
**Complexity:** Intermediate

**Architecture Overview:**
- **Algorithms:** Snowflake, UUID v4/v7, ULID, Database sequences
- **Constraints:** Uniqueness, ordering, performance, storage
- **Distributed:** No coordination or minimal coordination
- **Time-Based:** Chronological sorting support

**Key Components:**

**Snowflake (Twitter):**
1. **Structure (64 bits):**
   - 1 bit: Sign (always 0)
   - 41 bits: Timestamp (milliseconds since epoch)
   - 10 bits: Machine ID (datacenter + worker)
   - 12 bits: Sequence number
2. **Epoch:** Custom start time to extend range
3. **Sequence:** Rolls over every millisecond
4. **Clock Synchronization:** Handle clock drift

**UUID v7:**
1. **Structure (128 bits):**
   - 48 bits: Unix timestamp (milliseconds)
   - 4 bits: Version (7)
   - 12 bits: Random
   - 2 bits: Variant
   - 62 bits: Random
2. **Time-ordered:** Sortable by creation time
3. **No coordination:** Pure random after timestamp

**ULID:**
1. **Structure:**
   - 48 bits: Timestamp (milliseconds)
   - 80 bits: Randomness
2. **Lexicographically sortable:** String representation
3. **Base32 encoding:** 26 character string

**Design Patterns:**
- Factory Pattern (generator creation)
- Singleton Pattern (per-machine generator)
- Strategy Pattern (ID algorithm selection)

**Considerations:**
- Clock drift handling
- Machine ID assignment
- Sequence exhaustion
- Storage efficiency (binary vs string)

**Learning Outcomes:**
- Distributed unique ID generation
- Bit manipulation and packing
- Clock synchronization challenges
- Trade-offs between algorithms

---

## Phase 2: Concurrency Patterns (Go & TypeScript Mastery)

### 13-Worker-Pool
**Complexity:** Intermediate

**Architecture Overview:**
- **Pattern:** Fixed pool of workers processing jobs from shared queue
- **Components:** Job queue, worker goroutines/threads, dispatcher
- **Scalability:** Configurable pool size based on CPU cores
- **Graceful Shutdown:** Drain queue and stop workers cleanly

**Key Components:**
1. **Job Queue:** Buffered channel (Go) or Queue (TS) holding work items
2. **Worker Pool:** Fixed number of worker routines
3. **Dispatcher:** Routes jobs to available workers
4. **Worker Logic:**
   - Listen on job channel
   - Process job
   - Signal completion
5. **Result Channel:** Optional channel for results
6. **WaitGroup:** Track pending jobs

**Design Patterns:**
- Pool Pattern
- Producer-Consumer Pattern
- Observer Pattern (job completion)

**Advanced Features:**
- Dynamic pool resizing
- Worker health checks
- Priority queues
- Context cancellation propagation

**Learning Outcomes:**
- Goroutine/thread pool management
- Channel/queue communication patterns
- Graceful shutdown strategies
- Resource limiting techniques

---

### 14-Promise
**Complexity:** Intermediate

**Architecture Overview:**
- **States:** PENDING, FULFILLED, REJECTED
- **Chaining:** .then() and .catch() composition
- **Resolution:** Single resolution with immutable result
- **Concurrency:** Multiple callbacks registration

**Key Components:**
1. **Promise Structure:**
   - State (pending/fulfilled/rejected)
   - Value/Reason (resolution data)
   - Success callbacks array
   - Error callbacks array
2. **Executor Function:** Immediate execution with resolve/reject
3. **Then Method:** Register success callback, return new promise
4. **Catch Method:** Register error callback
5. **Resolve/Reject:** Transition state and invoke callbacks
6. **Chaining:** Each then() returns new promise for composition

**Design Patterns:**
- State Pattern
- Observer Pattern (callback registration)
- Chain of Responsibility (chaining)
- Monad Pattern (functional composition)

**Implementation Details:**
- Microtask queue for async resolution
- Error propagation through chains
- Promise.all() / Promise.race() utilities
- Finally blocks

**Learning Outcomes:**
- Asynchronous programming patterns
- State machine management
- Functional composition
- Callback vs Promise mental models

---

### 15-Semaphore
**Complexity:** Intermediate

**Architecture Overview:**
- **Purpose:** Control access to limited resources
- **Mechanism:** Counter + wait queue
- **Operations:** Acquire (wait), Release (signal)
- **Types:** Counting semaphore, Binary semaphore (mutex)

**Key Components:**
1. **Counter:** Current available permits
2. **Max Permits:** Total resource count
3. **Wait Queue:** Blocked acquire requests
4. **Acquire Operation:**
   - If counter > 0: decrement, proceed
   - Else: block/wait until available
5. **Release Operation:**
   - Increment counter
   - Wake waiting acquire (if any)

**Design Patterns:**
- Resource Pool Pattern
- Producer-Consumer (bounded buffer)

**Variants:**
- **Fair Semaphore:** FIFO ordering of acquires
- **Try-Acquire:** Non-blocking with timeout
- **Weighted Semaphore:** Different resource costs

**Learning Outcomes:**
- Resource limiting mechanisms
- Synchronization primitives
- Queue-based waiting
- Deadlock prevention

---

### 16-Read-Write-Lock
**Complexity:** Intermediate

**Architecture Overview:**
- **Pattern:** Multiple concurrent readers, exclusive writers
- **Starvation Prevention:** Writer priority or fair ordering
- **Recursion:** Reentrant locks support
- **Upgrade/Downgrade:** Convert between read and write

**Key Components:**
1. **State Tracking:**
   - Reader count
   - Writer active flag
   - Writer waiting count
2. **Lock Types:**
   - **Read Lock:** Acquire if no writer active/waiting
   - **Write Lock:** Acquire only when no readers or writers
3. **Condition Variables:** Reader/writer wait queues
4. **Fairness Modes:**
   - **Read Preference:** Readers don't block unless writer active
   - **Write Preference:** Writers get priority to prevent starvation
   - **Fair:** Strict FIFO ordering

**Design Patterns:**
- Lock Pattern
- Proxy Pattern (lock wrapper)

**Implementation Strategies:**
- **Standard:** Separate reader/writer counters + mutex
- **SeqLock:** Sequence counters for lock-free reads
- **RCU:** Read-Copy-Update for specialized use cases

**Learning Outcomes:**
- Concurrent read optimization
- Writer starvation handling
- Lock granularity trade-offs
- Performance vs fairness

---

### 17-Channel-Pool
**Complexity:** Intermediate

**Architecture Overview:**
- **Purpose:** Reuse channels for connection management
- **Lifecycle:** Acquire → Use → Release → Reset → Reuse
- **Capacity:** Fixed or dynamic pool sizing
- **Health Checks:** Validate channels before reuse

**Key Components:**
1. **Pool Structure:**
   - Available channels buffer
   - Active channel tracking
   - Factory function for new channels
2. **Acquire:**
   - Return available channel from pool
   - Create new if pool empty and under max
   - Block or error if at capacity
3. **Release:**
   - Reset/cleanup channel
   - Return to pool if healthy
   - Close and discard if unhealthy
4. **Channel Factory:** Creates properly configured channels

**Design Patterns:**
- Object Pool Pattern
- Factory Pattern

**Advanced Features:**
- Connection multiplexing
- Health check callbacks
- Idle timeout and cleanup
- Pool metrics (wait time, hit rate)

**Learning Outcomes:**
- Resource pooling strategies
- Channel lifecycle management
- Connection reuse patterns
- Pool sizing heuristics

---

### 18-Fan-Out-Fan-In
**Complexity:** Intermediate

**Architecture Overview:**
- **Pattern:** Distribute work to multiple workers, aggregate results
- **Stages:** Input → Fan Out (parallel workers) → Fan In (aggregator)
- **Synchronization:** Wait for all workers, collect results
- **Error Handling:** Partial failure management

**Key Components:**
1. **Input Stage:** Generate/provide work items
2. **Fan Out:**
   - Distribute work to N workers
   - Each worker processes subset
   - Workers run concurrently
3. **Worker Processing:**
   - Receive work
   - Process independently
   - Send result to aggregator
4. **Fan In:**
   - Collect results from all workers
   - Maintain ordering (optional)
   - Handle completion
5. **Aggregator:** Merge results into final output

**Design Patterns:**
- Pipeline Pattern
- Map-Reduce Pattern
- Scatter-Gather Pattern

**Implementation Approaches:**
- **Channels (Go):** Input channel, worker pool, result channel
- **Promises (TS):** Array of promises, Promise.all()
- **Generators:** Async iterators for streaming results

**Learning Outcomes:**
- Parallel processing patterns
- Result aggregation strategies
- Error handling in parallel flows
- Work distribution algorithms

---

### 19-Producer-Consumer
**Complexity:** Intermediate

**Architecture Overview:**
- **Pattern:** Decouple production from consumption rates
- **Buffer:** Bounded queue between producers and consumers
- **Coordination:** Blocking on full/empty buffer
- **Multiple Producers/Consumers:** Thread-safe queue required

**Key Components:**
1. **Buffer Queue:** Bounded-size thread-safe queue
2. **Producer:**
   - Generate items
   - Block when buffer full
   - Signal consumers when item added
3. **Consumer:**
   - Block when buffer empty
   - Process items
   - Signal producers when space available
4. **Synchronization:**
   - Mutex for queue access
   - Condition variables for blocking/waking
   - Or channels with buffering

**Design Patterns:**
- Producer-Consumer Pattern
- Blocking Queue Pattern
- Backpressure Pattern

**Variants:**
- **Multi-Producer-Multi-Consumer:** Lock-free queues
- **Priority Queue:** Priority-based consumption
- **Drop Policy:** Drop oldest/newest on overflow

**Learning Outcomes:**
- Rate decoupling strategies
- Bounded buffer management
- Condition variable usage
- Backpressure handling

---

### 20-Connection-Pool
**Complexity:** Intermediate

**Architecture Overview:**
- **Purpose:** Reuse expensive connections (DB, HTTP, etc.)
- **Lifecycle:** Create → Pool → Checkout → Use → Checkin → Validate
- **Sizing:** Min/max connections, dynamic scaling
- **Health:** Validation queries, idle timeout, max lifetime

**Key Components:**
1. **Pool State:**
   - Available connections (idle)
   - Active connections (checked out)
   - Factory for new connections
2. **Checkout:**
   - Return idle connection if available
   - Create new if under max and needed
   - Wait or fail if at capacity
3. **Checkin:**
   - Validate connection health
   - Reset state (transactions, session)
   - Return to pool or close if unhealthy
4. **Maintenance:**
   - Idle timeout cleanup
   - Max lifetime enforcement
   - Health check validation

**Design Patterns:**
- Object Pool Pattern
- Factory Pattern
- Proxy Pattern (connection wrapper)

**Configuration:**
- Min/Max pool size
- Connection timeout
- Validation query
- Idle timeout
- Max lifetime

**Learning Outcomes:**
- Connection lifecycle management
- Pool sizing strategies
- Resource cleanup and validation
- Performance optimization

---

## Phase 3: Storage Engines & Databases

### 21-B-Tree
**Complexity:** Advanced

**Architecture Overview:**
- **Structure:** Self-balancing search tree with variable node size
- **Order (m):** Max m-1 keys per node, max m children
- **Balance:** All leaves at same depth
- **Optimized for:** Disk I/O with large nodes (page-sized)

**Key Components:**
1. **Node Structure:**
   - Keys array (sorted)
   - Children pointers array
   - IsLeaf flag
   - Parent pointer (optional)
2. **B-Tree Properties:**
   - Every node has ≤ m children
   - Every internal node has ≥ ⌈m/2⌉ children
   - Root has ≥ 2 children (unless leaf)
   - All leaves at same depth
3. **Operations:**
   - **Search:** Binary search keys, traverse to child
   - **Insert:** Add to leaf, split if overflow, propagate up
   - **Delete:** Remove, borrow from sibling or merge, propagate
   - **Split:** Divide node, promote middle key
   - **Merge/Redistribute:** Handle underflow

**Design Patterns:**
- Composite Pattern (tree structure)
- Iterator Pattern (range scans)

**Optimizations:**
- B+ Tree variant (keys only in leaves, linked leaves)
- Prefix compression
- Bulk loading

**Learning Outcomes:**
- Self-balancing tree mechanics
- Disk-optimized data structures
- Node splitting/merging algorithms
- Range query optimization

---

### 22-LSM-Tree (Log-Structured Merge Tree)
**Complexity:** Advanced

**Architecture Overview:**
- **Write Pattern:** Sequential writes to immutable files
- **Structure:** MemTable → WAL → SSTables (sorted files)
- **Compaction:** Background merging of SSTables
- **Read Path:** Check memtable → bloom filters → SSTables

**Key Components:**
1. **Write Path:**
   - **WAL (Write-Ahead Log):** Durability, append-only
   - **MemTable:** In-memory sorted structure (skip list or B-tree)
   - **Flush:** MemTable → Immutable SSTable on disk
2. **Storage Layers:**
   - **Level 0:** Recently flushed SSTables (can overlap)
   - **Level 1+:** Sorted, non-overlapping SSTables
   - **Compaction:** Merge SSTables, remove duplicates/tombstones
3. **SSTable Format:**
   - Data blocks (sorted key-value pairs)
   - Index block (sparse index)
   - Filter block (bloom filters)
   - Footer (metadata)
4. **Read Path:**
   - Check MemTable
   - Check Immutable MemTables
   - Query SSTables (newest to oldest)
   - Use bloom filters to skip SSTables

**Compaction Strategies:**
- **Size-Tiered:** Merge similar-sized SSTables
- **Leveled:** Fixed-size levels, promote on compaction
- **Tiered+Leveled:** Hybrid approach

**Design Patterns:**
- Write-Ahead Logging Pattern
- Immutable Data Pattern
- Compaction Pattern

**Learning Outcomes:**
- Sequential write optimization
- Immutable data structures
- Compaction algorithms
- Read amplification trade-offs

---

### 23-Write-Ahead-Log (WAL)
**Complexity:** Intermediate

**Architecture Overview:**
- **Purpose:** Durability, crash recovery, replication
- **Format:** Append-only log of operations
- **Sync Strategy:** Buffer, sync interval, or sync every write
- **Recovery:** Replay log to reconstruct state

**Key Components:**
1. **Log Entry Structure:**
   - Sequence number (LSN)
   - Operation type (INSERT/UPDATE/DELETE)
   - Key, value, timestamp
   - Checksum for integrity
2. **Log Management:**
   - Append new entries
   - Periodic checkpointing (truncate log)
   - Segmented log files
3. **Durability Levels:**
   - **Async:** Buffer writes, periodic fsync
   - **Sync:** fsync on every commit
   - **Group Commit:** Batch multiple commits
4. **Recovery Process:**
   - Read log from last checkpoint
   - Replay operations to rebuild state
   - Verify checksums
   - Apply to storage engine

**Design Patterns:**
- Command Pattern (log entries as commands)
- Event Sourcing Pattern

**Formats:**
- **Binary:** Efficient, compact
- **JSON/Proto:** Human-readable, schema evolution

**Learning Outcomes:**
- Durability guarantees
- Crash recovery mechanisms
- Sequential I/O optimization
- Log truncation strategies

---

### 24-Inverted-Index
**Complexity:** Intermediate

**Architecture Overview:**
- **Purpose:** Full-text search, term → document mapping
- **Structure:** Term dictionary + Postings lists
- **Compression:** Delta encoding, variable byte encoding
- **Ranking:** TF-IDF, BM25 scoring

**Key Components:**
1. **Term Dictionary:**
   - Sorted list of unique terms
   - Term → postings list pointer
   - Prefix compression (front coding)
2. **Postings List:**
   - List of document IDs containing term
   - Document frequency
   - Optional: positions, term frequency
3. **Indexing Process:**
   - Tokenization (split text into terms)
   - Normalization (lowercase, stemming)
   - Build postings lists
   - Sort and compress
4. **Query Processing:**
   - Parse query (AND, OR, NOT)
   - Fetch postings lists
   - Merge lists (intersection/union)
   - Score and rank results

**Compression Techniques:**
- **Delta Encoding:** Store gaps between doc IDs
- **Variable Byte:** Smaller encoding for small numbers
- **Roaring Bitmaps:** Compressed bitmaps for boolean queries

**Design Patterns:**
- Inverted Index Pattern
- Tokenizer Pattern
- Scoring Strategy Pattern

**Learning Outcomes:**
- Text indexing algorithms
- Compression techniques
- Boolean query processing
- Relevance scoring

---

### 25-Time-Series-Store
**Complexity:** Advanced

**Architecture Overview:**
- **Data Model:** Timestamp + metric name + tags + value
- **Write Pattern:** High-volume, ordered by time
- **Storage:** Time-partitioned, compressed chunks
- **Query Pattern:** Time range scans, aggregations

**Key Components:**
1. **Data Layout:**
   - **Time Partitioning:** Data organized by time windows
   - **Series:** Unique combination of metric + tags
   - **Chunks:** Compressed blocks of time-ordered data
2. **Storage Format:**
   - **Timestamp Compression:** Delta-of-delta encoding
   - **Value Compression:** XOR-based floating point (Gorilla)
   - **String Deduplication:** Tag value dictionary
3. **Indexing:**
   - Series index: Tag → series IDs
   - Time index: Series + time → chunk location
4. **Retention:**
   - Automatic expiration of old data
   - Downsampling for long-term retention

**Compression Algorithms:**
- **Gorilla:** XOR-based floating point compression
- **Delta Encoding:** For timestamps
- **Dictionary Encoding:** For string tags

**Design Patterns:**
- Time Series Pattern
- Columnar Storage Pattern
- Downsampling Pattern

**Learning Outcomes:**
- Time-series data modeling
- Compression for time-ordered data
- Partitioning strategies
- High-throughput ingestion

---

### 26-Key-Value-Store
**Complexity:** Advanced

**Architecture Overview:**
- **Storage Engine:** LSM-Tree or B-Tree based
- **Interface:** Get, Put, Delete, Range Scan
- **Features:** Transactions, snapshots, TTL
- **Distribution:** Optional sharding support

**Key Components:**
1. **Storage Layer:**
   - LSM-Tree or B-Tree backend
   - Write-Ahead Log for durability
   - Block cache for hot data
2. **API Layer:**
   - CRUD operations
   - Batch writes
   - Snapshots (point-in-time reads)
3. **Memory Management:**
   - MemTable for recent writes
   - Block cache for SSTables
   - Memory limits and flush triggers
4. **Advanced Features:**
   - TTL (Time To Live) for automatic expiration
   - Compression (Snappy, LZ4, Zstd)
   - Checksums for data integrity

**Design Patterns:**
- Storage Engine Pattern
- Snapshot Isolation Pattern
- MVCC Pattern

**Learning Outcomes:**
- KV store internals
- Storage engine integration
- Transaction support
- Performance optimization

---

### 27-Document-Store
**Complexity:** Advanced

**Architecture Overview:**
- **Data Model:** JSON-like documents with flexible schema
- **Indexing:** Secondary indexes on document fields
- **Query Language:** Document query interface (MongoDB-like)
- **Storage:** BSON/JSON binary encoding

**Key Components:**
1. **Document Model:**
   - Document ID (primary key)
   - JSON/BSON payload
   - Metadata (version, timestamps)
2. **Storage Layer:**
   - KV store backend (document ID → document)
   - Index storage (field value → document IDs)
3. **Indexing:**
   - Secondary indexes on fields
   - Compound indexes (multiple fields)
   - Index types: B-tree, Hash, Text
4. **Query Engine:**
   - Query parser
   - Index selection
   - Document fetching and filtering
   - Projection (field selection)

**Design Patterns:**
- Document Store Pattern
- Secondary Index Pattern
- Query Planner Pattern

**Learning Outcomes:**
- Schema-less data modeling
- Secondary index management
- Query optimization
- Document storage formats

---

### 28-Graph-Database
**Complexity:** Advanced

**Architecture Overview:**
- **Data Model:** Nodes, edges, properties
- **Storage:** Adjacency list or matrix
- **Query:** Graph traversal patterns
- **Algorithms:** Shortest path, centrality, community detection

**Key Components:**
1. **Graph Model:**
   - **Nodes:** Entities with properties
   - **Edges:** Relationships (directed/undirected) with properties
   - **Properties:** Key-value pairs on nodes/edges
2. **Storage Layout:**
   - **Adjacency List:** Node → [edges] mapping
   - **Index-Free Adjacency:** Direct pointer to connected nodes
   - **Property Store:** Separate storage for large properties
3. **Query Interface:**
   - Pattern matching (Cypher, Gremlin-like)
   - Traversal API
   - Graph algorithms library
4. **Traversal Engine:**
   - DFS, BFS
   - Shortest path (Dijkstra, A*)
   - Pattern matching

**Design Patterns:**
- Graph Pattern
- Property Graph Pattern
- Traversal Pattern

**Learning Outcomes:**
- Graph data modeling
- Adjacency list optimization
- Graph query languages
- Traversal algorithms

---

### 29-Geospatial-Index
**Complexity:** Advanced

**Architecture Overview:**
- **Spatial Data:** Points, lines, polygons with coordinates
- **Indexing:** R-Tree, Quadtree, Geohash, S2, H3
- **Queries:** Range, nearest neighbor, intersection, containment
- **Projections:** Latitude/longitude to flat coordinates

**Key Components:**

**R-Tree:**
1. **Structure:** Balanced tree with bounding boxes
2. **Node:** Minimum bounding rectangle (MBR) covering children
3. **Operations:**
   - **Insert:** Find best leaf, add entry, split if overflow
   - **Search:** Recursively check MBR intersection
   - **Delete:** Remove, reinsert if underflow
4. **Variants:** R*Tree, R+Tree (reduced overlap)

**Quadtree:**
1. **Structure:** Recursive spatial subdivision
2. **Subdivision:** Split cell into 4 quadrants when threshold exceeded
3. **Depth:** Adaptive based on data density
4. **Queries:** Navigate tree based on query region

**Geohash:**
1. **Encoding:** Interleave lat/lon bits, base32 encode
2. **Precision:** Longer hash = smaller region
3. **Neighboring:** Adjacent cells share prefix
4. **Query:** Hash prefix matching for range queries

**Design Patterns:**
- Spatial Index Pattern
- Bounding Volume Hierarchy

**Learning Outcomes:**
- Spatial data structures
- Bounding box hierarchies
- Coordinate systems and projections
- Nearest neighbor algorithms

---

### 30-Column-Store
**Complexity:** Advanced

**Architecture Overview:**
- **Data Layout:** Column-oriented (vs row-oriented)
- **Compression:** Better compression ratio per column
- **Projection:** Read only needed columns
- **Analytics:** Optimized for OLAP workloads

**Key Components:**
1. **Columnar Storage:**
   - Each column stored separately
   - Same-type data = better compression
   - Vectorized processing
2. **Row Group:**
   - Horizontal partition of rows
   - Contains column chunks for each column
   - Typical size: 100K-1M rows
3. **Column Chunk:**
   - Compressed data for one column in row group
   - Metadata: min/max values, null counts, encoding
4. **Metadata:**
   - Schema
   - Row group locations
   - Column statistics for pruning

**Compression:**
- **Run-Length Encoding:** Repeated values
- **Dictionary Encoding:** Low cardinality columns
- **Delta Encoding:** Time series, sorted data
- **Bit Packing:** Small integer ranges

**Design Patterns:**
- Columnar Storage Pattern
- Predicate Pushdown Pattern

**Learning Outcomes:**
- Column vs row storage trade-offs
- Vectorized execution
- Compression algorithms
- Analytical query optimization

---

## Phase 4: Distributed Systems (The Real Challenge)

### 31-Leader-Election
**Complexity:** Advanced

**Architecture Overview:**
- **Goal:** Single coordinator in distributed system
- **Algorithms:** Bully algorithm, Raft, ZooKeeper (ZAB), etcd
- **Safety:** At most one leader at a time
- **Liveness:** Leader elected eventually

**Key Components (Bully Algorithm):**
1. **Node State:**
   - ID (unique, comparable)
   - Role: Leader, Follower, Candidate
2. **Election Trigger:**
   - Leader failure detection (timeout)
   - Startup
3. **Election Process:**
   - Send ELECTION to higher-ID nodes
   - Wait for OK responses
   - If no OKs → become leader
   - If OK → wait for COORDINATOR message
4. **Coordinator Message:**
   - New leader announces leadership
   - Nodes update leader reference

**Key Components (Raft - simplified):**
1. **States:** Follower, Candidate, Leader
2. **Term:** Monotonically increasing epoch
3. **Election Timeout:** Randomized to prevent split votes
4. **Voting:** Grant vote to first candidate with higher/equal term
5. **Log Matching:** Leader must have most up-to-date log

**Design Patterns:**
- Leader-Follower Pattern
- Consensus Pattern
- Failure Detector Pattern

**Learning Outcomes:**
- Consensus fundamentals
- Failure detection
- Split-brain prevention
- Term/epoch concepts

---

### 32-Distributed-Lock
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Mutual exclusion across processes/machines
- **Properties:** Safety (only one holder), Liveness (no deadlock), Fault-tolerance
- **Backends:** Redis (Redlock), ZooKeeper, etcd, Consul
- **Lease-Based:** Expiration prevents indefinite locks

**Key Components:**
1. **Lock Structure:**
   - Resource identifier
   - Token/UUID (fencing token)
   - Expiry/lease duration
   - Owner identifier
2. **Acquire Operation:**
   - Atomic compare-and-set
   - Set only if not exists or expired
   - Include fencing token
3. **Release Operation:**
   - Verify ownership (token match)
   - Atomic delete
   - Auto-expire as safety net
4. **Fencing Tokens:**
   - Monotonically increasing
   - Protect against delayed releases
   - Storage validates token on write

**Redlock Algorithm (Redis):**
1. Acquire lock on N/2+1 Redis instances
2. Use same key and unique value
3. Acquire timeout must be < lock validity
4. If acquired majority → lock held
5. Release on all instances

**Design Patterns:**
- Lock Pattern (distributed variant)
- Lease Pattern
- Fencing Pattern

**Learning Outcomes:**
- Distributed mutual exclusion
- Clock skew handling
- Fencing token importance
- Lease-based safety

---

### 33-Gossip-Protocol
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Propagate state in distributed system
- **Mechanism:** Epidemic broadcast, random peer selection
- **Properties:** Scalable, fault-tolerant, eventually consistent
- **Use Cases:** Membership, failure detection, state replication

**Key Components:**
1. **Node State:**
   - Local state table
   - Version vectors or timestamps
   - Heartbeat counters
2. **Gossip Messages:**
   - **Push:** Send local state to random peer
   - **Pull:** Request state from random peer
   - **Push-Pull:** Exchange both ways
3. **Peer Selection:**
   - Random selection (uniform)
   - Biased (prefer known peers)
   - Exponential peer discovery
4. **Convergence:**
   - O(log N) rounds for full propagation
   - Anti-entropy for reconciliation

**Algorithms:**
- **Rumor Mongering:** Push only, stop after k rounds
- **Anti-Entropy:** Periodic full state comparison
- **SWIM:** Scalable weakly-consistent infection-style membership

**Design Patterns:**
- Gossip Pattern
- Epidemic Broadcast Pattern
- Eventually Consistent Pattern

**Learning Outcomes:**
- Epidemic algorithms
- Scalable dissemination
- Eventual consistency
- Failure detection with gossip

---

### 34-Vector-Clocks
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Track causality in distributed systems
- **Mechanism:** Vector of counters, one per node
- **Ordering:** Happens-before relationship detection
- **Conflict Detection:** Concurrent updates identification

**Key Components:**
1. **Vector Clock Structure:**
   - Map: NodeID → Counter
   - Increment local counter on event
   - Merge clocks on receive
2. **Operations:**
   - **Increment:** VC[node]++ on local event
   - **Merge:** VC[node] = max(VC[node], received[node])
   - **Compare:** Determine ordering or concurrency
3. **Comparison Rules:**
   - VC1 < VC2: if all VC1[i] ≤ VC2[i] and at least one <
   - VC1 || VC2: concurrent (neither < other)
4. **Versioning:**
   - Attach clock to every update
   - Track causal dependencies

**Optimizations:**
- **Dotted Version Vectors:** Single counter for writer
- **Interval Tree Clocks:** Dynamic node sets
- **Pruning:** Remove entries for removed nodes

**Design Patterns:**
- Version Vector Pattern
- Causal Consistency Pattern

**Learning Outcomes:**
- Causality tracking
- Partial ordering
- Conflict resolution
- Happens-before relation

---

### 35-CRDT (Conflict-free Replicated Data Types)
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Replicated data without coordination
- **Properties:** Strong eventual consistency, automatic conflict resolution
- **Types:** State-based (CvRDT), Operation-based (CmRDT)
- **Use Cases:** Collaborative editing, distributed counters, shopping carts

**Key Components:**
1. **CRDT Properties:**
   - **Associative:** (a⊕b)⊕c = a⊕(b⊕c)
   - **Commutative:** a⊕b = b⊕a
   - **Idempotent:** a⊕a = a
2. **State-Based (CvRDT):**
   - Merge function combines states
   - Monotonic semilattice
   - Payload grows monotonically
3. **Operation-Based (CmRDT):**
   - Broadcast operations
   - Operations must commute
   - Causal broadcast required

**Common CRDTs:**
- **G-Counter:** Grow-only counter (vector of increments)
- **PN-Counter:** Positive-negative counter (two G-Counters)
- **G-Set:** Grow-only set
- **LWW-Register:** Last-write-wins register (timestamped value)
- **OR-Set:** Observed-removed set (unique tags for elements)
- **RGA:** Replicated growable array (for collaborative text)

**Design Patterns:**
- CRDT Pattern
- Commutative Replication Pattern

**Learning Outcomes:**
- Conflict-free replication
- Strong eventual consistency
- Commutative operations
- State merging strategies

---

### 36-Distributed-Transaction (Saga Pattern)
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Long-running transactions across services
- **Approach:** Sequence of local transactions with compensations
- **Coordination:** Choreography (event-driven) or Orchestration (central coordinator)
- **Consistency:** Eventual consistency, compensating actions

**Key Components:**
1. **Saga:**
   - Sequence of steps
   - Each step: local transaction
   - Compensating action for rollback
2. **Choreography:**
   - Services listen to events
   - Each service executes step, publishes next event
   - Decentralized, loosely coupled
3. **Orchestration:**
   - Central coordinator (saga orchestrator)
   - Orchestrator calls services in sequence
   - Compensates on failure
4. **Compensation:**
   - Undo completed steps on failure
   - Idempotent compensations
   - Must succeed (or require manual intervention)

**Design Patterns:**
- Saga Pattern
- Compensation Pattern
- Orchestration/Choreography Pattern

**Example Flow (Order Saga):**
1. Create Order → 2. Reserve Payment → 3. Reserve Inventory → 4. Ship Order
- If #3 fails: Compensate #2, Compensate #1

**Learning Outcomes:**
- Long-running transactions
- Compensating transactions
- Event-driven coordination
- Eventual consistency in business logic

---

### 37-Two-Phase-Commit (2PC)
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Atomic commit across multiple resources
- **Phases:** Prepare (voting), Commit (execution)
- **Roles:** Coordinator, Participants
- **Properties:** ACID across distributed resources

**Key Components:**
1. **Coordinator:**
   - Manages transaction
   - Sends prepare/commit/abort
   - Records decision in durable log
2. **Participants:**
   - Execute local prepare
   - Vote YES (prepared) or NO
   - Commit or abort on coordinator decision
3. **Phase 1 (Prepare):**
   - Coordinator sends PREPARE to all
   - Participants prepare (acquire locks, write undo/redo)
   - Participants vote YES/NO
4. **Phase 2 (Commit/Abort):**
   - If all YES → send COMMIT
   - If any NO → send ABORT
   - Participants execute decision

**Failure Handling:**
- **Coordinator Failure:** Participants block until recovery
- **Participant Failure:** Coordinator decides based on votes
- **Recovery:** Coordinator log replay, participant inquiry

**Optimizations:**
- **Three-Phase Commit (3PC):** Non-blocking under certain assumptions
- **Presumed Abort/Commit:** Reduce logging overhead

**Design Patterns:**
- Two-Phase Commit Pattern
- Transaction Coordinator Pattern

**Learning Outcomes:**
- Distributed atomic commit
- Blocking vs non-blocking protocols
- Coordinator failure recovery
- ACID trade-offs in distributed systems

---

### 38-Replication
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Fault tolerance, read scaling, availability
- **Models:** Single-leader, Multi-leader, Leaderless
- **Consistency:** Sync vs Async, eventual consistency
- **Failover:** Automatic leader promotion

**Key Components:**

**Single-Leader (Primary-Replica):**
1. **Leader:** Accepts all writes, propagates to replicas
2. **Followers:** Apply changes from leader
3. **Replication Log:**
   - **Statement-based:** Replay SQL statements
   - **Write-ahead log:** Ship WAL
   - **Row-based:** Log row changes
   - **Trigger-based:** Application-level changes
4. **Sync Modes:**
   - **Synchronous:** Wait for replica ack (durability, latency)
   - **Asynchronous:** Fire-and-forget (performance, lag)
   - **Semi-sync:** Wait for N replicas

**Multi-Leader:**
- Multiple leaders accept writes
- Conflict resolution needed
- Use case: Multi-datacenter, offline clients

**Leaderless (Dynamo-style):**
- Writes go to N replicas
- Read repair or anti-entropy for consistency
- Quorum reads/writes (W + R > N)

**Design Patterns:**
- Primary-Replica Pattern
- Replication Pattern
- Failover Pattern

**Learning Outcomes:**
- Replication strategies
- Consistency vs availability trade-offs
- Replication lag handling
- Failover mechanics

---

### 39-Sharding
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Horizontal scaling, data partitioning
- **Strategy:** Key-based, Range-based, Hash-based
- **Routing:** Application-level, Proxy, Coordinator
- **Rebalancing:** Moving shards as data grows

**Key Components:**
1. **Sharding Strategies:**
   - **Hash-based:** hash(key) % num_shards
   - **Range-based:** Key ranges per shard (A-M, N-Z)
   - **Directory-based:** Lookup table mapping
2. **Shard Key Selection:**
   - Cardinality (enough distinct values)
   - Access patterns (even distribution)
   - Avoid hotspots (time-based keys)
3. **Routing Layer:**
   - Application logic
   - Proxy/gateway
   - Consistent hashing ring
4. **Rebalancing:**
   - Split hot shards
   - Migrate data with dual-write
   - Update routing

**Challenges:**
- **Cross-shard queries:** Scatter-gather, denormalization
- **Joins:** Avoid or application-side joins
- **Transactions:** Limited to single shard (or 2PC)
- **Auto-sharding:** Automatic split/merge

**Design Patterns:**
- Sharding Pattern
- Partitioning Pattern
- Scatter-Gather Pattern

**Learning Outcomes:**
- Data partitioning strategies
- Hotspot avoidance
- Rebalancing techniques
- Cross-shard operation handling

---

### 40-Consensus (Raft - Simplified)
**Complexity:** Expert

**Architecture Overview:**
- **Purpose:** Agree on value/state across distributed nodes
- **Algorithm:** Raft (understandable consensus)
- **Properties:** Safety, availability with majority
- **Components:** Leader election, log replication, safety

**Key Components:**
1. **Server States:**
   - **Leader:** Handle all client requests, replicate log
   - **Follower:** Passive, respond to RPCs
   - **Candidate:** Seeking election
2. **Term:** Logical clock, monotonically increasing
3. **Leader Election:**
   - Timeout → increment term → become candidate
   - Request votes from peers
   - Win if majority votes
4. **Log Replication:**
   - Client requests appended to leader log
   - Leader sends AppendEntries RPCs
   - Entries committed when replicated to majority
5. **Safety:**
   - Election restriction (up-to-date log)
   - Commit rules (current term entries)
   - Leader completeness

**RPCs:**
- **RequestVote:** Candidate requests votes
- **AppendEntries:** Heartbeat + log replication

**Design Patterns:**
- Consensus Pattern
- Replicated State Machine Pattern
- Leader Election Pattern

**Learning Outcomes:**
- Consensus fundamentals
- Log replication
- Safety guarantees
- Failure handling in consensus

---

### 41-Service-Discovery
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Locate services dynamically
- **Pattern:** Registry-based (client-side or server-side discovery)
- **Backends:** Consul, etcd, ZooKeeper, Eureka
- **Health Checks:** Automatic deregistration of unhealthy instances

**Key Components:**
1. **Service Registry:**
   - Central database of service instances
   - Instance ID, host, port, metadata
   - Health status
2. **Registration:**
   - Self-registration (service registers itself)
   - Third-party registration (agent/orchestrator)
3. **Discovery:**
   - Client queries registry
   - Receives list of healthy instances
   - Caches with TTL
4. **Health Checking:**
   - Heartbeats from services
   - Active health checks by registry
   - Automatic deregistration

**Client-Side Discovery:**
- Client queries registry, chooses instance
- Load balancing in client

**Server-Side Discovery:**
- Client talks to load balancer
- Load balancer queries registry

**Design Patterns:**
- Service Registry Pattern
- Discovery Pattern
- Health Check Pattern

**Learning Outcomes:**
- Dynamic service location
- Registry patterns
- Health checking strategies
- Client vs server-side discovery

---

### 42-Load-Balancer
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Distribute traffic across multiple servers
- **Layer:** L4 (transport) or L7 (application)
- **Algorithms:** Round-robin, Least connections, IP hash, weighted
- **Health Checks:** Detect and remove failed backends

**Key Components:**
1. **Frontend:**
   - Listen on VIP (virtual IP)
   - Accept client connections
   - TLS termination (L7)
2. **Backend Pool:**
   - List of healthy servers
   - Health status tracking
   - Dynamic addition/removal
3. **Load Balancing Algorithms:**
   - **Round Robin:** Sequential distribution
   - **Least Connections:** Route to least loaded
   - **IP Hash:** Consistent hashing on client IP
   - **Weighted:** Capacity-based distribution
   - **Random:** With or without weighting
4. **Health Checks:**
   - Active (ping, HTTP check)
   - Passive (monitor responses)
   - Mark unhealthy, redistribute traffic

**Session Persistence:**
- Sticky sessions (IP hash, cookie)
- Shared session storage

**Design Patterns:**
- Load Balancer Pattern
- Reverse Proxy Pattern
- Health Check Pattern

**Learning Outcomes:**
- Traffic distribution algorithms
- Health check implementation
- Session persistence
- High availability patterns

---

## Phase 5: Real-World Systems (Complete Applications)

### 43-URL-Shortener
**Complexity:** Intermediate

**Architecture Overview:**
- **Core Function:** Map long URLs to short codes
- **Encoding:** Base62 encoding of sequential or hash-based IDs
- **Storage:** Key-value store (short code → long URL)
- **Analytics:** Click tracking, referrer, geographic data

**Key Components:**
1. **URL Shortening:**
   - Accept long URL
   - Generate unique ID (Snowflake, hash, counter)
   - Encode to Base62 (a-z, A-Z, 0-9)
   - Store mapping
2. **URL Resolution:**
   - Decode short code to ID
   - Lookup in KV store
   - 301/302 redirect to long URL
3. **Collision Handling:**
   - Check if code exists
   - Regenerate or append counter on collision
4. **Analytics:**
   - Async click logging
   - Time-series aggregation
   - Dashboard API
5. **Caching:**
   - Redis cache for hot URLs
   - Browser caching (Cache-Control headers)

**Design Decisions:**
- **Code length:** Balance uniqueness vs brevity (6-8 chars)
- **ID generation:** Sequential (predictable) vs random (secure)
- **Expiration:** TTL for unused URLs
- **Custom aliases:** User-defined short codes

**Design Patterns:**
- Redirect Pattern
- Encoding Pattern
- Analytics Pipeline Pattern

**Learning Outcomes:**
- Encoding schemes
- KV store design
- Redirect optimization
- Analytics pipeline

---

### 44-Message-Queue
**Complexity:** Advanced

**Architecture Overview:**
- **Model:** FIFO queues with publish/subscribe semantics
- **Delivery:** At-least-once (default), at-most-once, exactly-once
- **Persistence:** Disk-backed for durability
- **Scalability:** Partitioned topics, consumer groups

**Key Components:**
1. **Message Model:**
   - **Queue:** Point-to-point delivery
   - **Topic:** Pub/sub broadcast
   - **Partition:** Ordered subset of topic
2. **Producer:**
   - Publish messages to topic/queue
   - ACK on receipt or replication
   - Retry on failure
3. **Broker:**
   - Receive and store messages
   - Manage partitions
   - Handle replication
4. **Consumer:**
   - Subscribe to topics
   - Pull or push delivery
   - Acknowledge processing
5. **Storage:**
   - Log-based (append-only)
   - Segment files with index
   - Compaction for key-value topics

**Delivery Semantics:**
- **At-most-once:** Fire and forget
- **At-least-once:** Retry until ACK, possible duplicates
- **Exactly-once:** Idempotent producers + transactions

**Advanced Features:**
- **Consumer Groups:** Parallel processing with partition assignment
- **Dead Letter Queue:** Failed message handling
- **Delayed Messages:** Scheduled delivery
- **Message TTL:** Automatic expiration

**Design Patterns:**
- Message Queue Pattern
- Pub/Sub Pattern
- Consumer Group Pattern

**Learning Outcomes:**
- Queue-based architectures
- Delivery semantics
- Partitioning strategies
- Log-based storage

---

### 45-WebSocket-Server
**Complexity:** Intermediate

**Architecture Overview:**
- **Protocol:** Full-duplex over single TCP connection
- **Handshake:** HTTP upgrade to WebSocket
- **Messaging:** Frame-based, binary/text support
- **Scaling:** Sticky sessions or pub/sub backend

**Key Components:**
1. **Connection Management:**
   - HTTP upgrade handling
   - Connection state tracking
   - Heartbeat/ping-pong
2. **Frame Protocol:**
   - **FIN:** Final fragment flag
   - **Opcode:** Text, binary, close, ping, pong
   - **Mask:** Client-to-server masking
   - **Payload:** Up to 64-bit length
3. **Message Handling:**
   - Frame parsing and assembly
   - Text encoding (UTF-8 validation)
   - Binary pass-through
4. **Broadcasting:**
   - Room/channel subscriptions
   - Fan-out to connected clients
   - Efficient message routing
5. **Scaling:**
   - Redis pub/sub for multi-server
   - Consistent hashing for room assignment

**Connection Lifecycle:**
1. Client sends HTTP upgrade request
2. Server responds with 101 Switching Protocols
3. Bidirectional frame exchange
4. Close handshake (or connection drop)

**Design Patterns:**
- Observer Pattern (event handlers)
- Room Pattern (group management)
- Heartbeat Pattern

**Learning Outcomes:**
- WebSocket protocol
- Frame parsing
- Connection state management
- Real-time broadcast scaling

---

### 46-Chat-System
**Complexity:** Advanced

**Architecture Overview:**
- **Model:** WhatsApp-style messaging
- **Features:** 1:1 chat, groups, read receipts, media, presence
- **Storage:** Message history, user metadata
- **Delivery:** Real-time + offline queue

**Key Components:**
1. **User Management:**
   - Authentication (JWT, session)
   - Presence status (online/last seen)
   - Profile and contacts
2. **Messaging:**
   - Message structure (ID, sender, recipient, content, timestamp)
   - Persistent message queue per user
   - Delivery receipts (sent, delivered, read)
3. **Real-Time Delivery:**
   - WebSocket for online users
   - Push notifications for offline
   - Message sync on reconnect
4. **Group Chat:**
   - Group metadata (members, admins)
   - Fan-out on send (write to all member queues)
   - Read receipts per member
5. **Media Storage:**
   - Object storage (S3-like) for images/files
   - Thumbnail generation
   - CDN for delivery
6. **Database Schema:**
   - **Users:** Profile, credentials, settings
   - **Conversations:** 1:1 or group
   - **Messages:** Content, metadata, status
   - **Message Status:** Per-recipient delivery state

**Scaling Considerations:**
- **Fan-out:** Group messages → write to N queues
- **Sharding:** By user_id or conversation_id
- **Caching:** Hot conversations in Redis
- **Read receipts:** Aggregate updates, periodic sync

**Design Patterns:**
- Inbox Pattern (per-user queue)
- Fan-Out Pattern
- Receipt Pattern

**Learning Outcomes:**
- Messaging architecture
- Fan-out vs fan-in strategies
- Delivery status tracking
- Real-time + offline hybrid

---

### 47-Task-Scheduler
**Complexity:** Intermediate

**Architecture Overview:**
- **Purpose:** Execute tasks at specific times or intervals
- **Types:** One-time (delayed), Recurring (cron), Distributed
- **Precision:** Second-level or minute-level
- **Reliability:** At-least-once execution

**Key Components:**
1. **Task Definition:**
   - Task ID, payload, execute_at time
   - Recurrence rule (cron expression)
   - Retry policy
   - Timeout
2. **Scheduling:**
   - **Min-Heap:** Priority queue by execute time
   - **Time Wheel:** Buckets for efficient lookup
   - **Database:** Polling with index on time
3. **Execution:**
   - Worker pool picks ready tasks
   - Execute job (HTTP call, function, etc.)
   - Handle success/failure
   - Reschedule recurring tasks
4. **Persistence:**
   - Task queue in database
   - Execution history
   - State machine (pending, running, completed, failed)
5. **Distributed Coordination:**
   - Leader election for scheduler
   - Lock tasks before execution
   - Prevent duplicate execution

**Cron Expression Parsing:**
- Minute, Hour, Day of Month, Month, Day of Week
- Special characters: * (all), / (step), - (range), , (list)
- Next execution time calculation

**Design Patterns:**
- Scheduler Pattern
- Job Queue Pattern
- Cron Pattern

**Learning Outcomes:**
- Time-based job scheduling
- Cron expression parsing
- Distributed task coordination
- Delayed queue implementation

---

### 48-Search-Engine
**Complexity:** Advanced

**Architecture Overview:**
- **Pipeline:** Crawl → Index → Query → Rank
- **Indexing:** Inverted index with TF-IDF/BM25 scoring
- **Querying:** Boolean, phrase, fuzzy matching
- **Scaling:** Distributed indexing and querying

**Key Components:**
1. **Crawler (Optional):**
   - URL frontier
   - Politeness (rate limiting)
   - Content extraction
   - Deduplication (URL + content hash)
2. **Indexer:**
   - Tokenization and normalization
   - Build inverted index (term → doc list)
   - Store document content
   - Build auxiliary indexes (suggest, spellcheck)
3. **Query Engine:**
   - Query parsing (AND, OR, NOT, phrases)
   - Index lookup
   - Boolean expression evaluation
   - Scoring (TF-IDF, BM25, PageRank)
4. **Ranking:**
   - Term frequency (TF)
   - Inverse document frequency (IDF)
   - Field boosts (title > body)
   - Recency, popularity signals
5. **Distributed Search:**
   - Index sharding by document
   - Scatter-gather queries
   - Result merging and reranking

**Query Types:**
- **Keyword:** Simple term search
- **Phrase:** Exact match "foo bar"
- **Boolean:** AND, OR, NOT
- **Wildcard:** prefix*, *suffix
- **Fuzzy:** Approximate match

**Design Patterns:**
- Inverted Index Pattern
- Crawler Pattern
- Scatter-Gather Pattern

**Learning Outcomes:**
- Full-text indexing
- Query processing
- Relevance scoring
- Distributed search architecture

---

### 49-API-Gateway
**Complexity:** Advanced

**Architecture Overview:**
- **Purpose:** Single entry point for microservices
- **Responsibilities:** Routing, auth, rate limiting, transformation
- **Architecture:** Reverse proxy with middleware chain
- **Performance:** Connection pooling, caching

**Key Components:**
1. **Routing:**
   - Path-based routing (/users → user-service)
   - Host-based routing (api.example.com)
   - Parameter-based routing
   - Rewrite rules
2. **Middleware Chain:**
   - Authentication (JWT, API key validation)
   - Rate limiting (token bucket per client)
   - Request/Response transformation
   - Logging and metrics
3. **Load Balancing:**
   - Service discovery integration
   - Health check-based routing
   - Circuit breaker integration
4. **Caching:**
   - Response caching with TTL
   - Cache key generation
   - Invalidation strategies
5. **Protocol Translation:**
   - HTTP ↔ gRPC
   - WebSocket support
   - GraphQL federation

**Middleware Pipeline:**
```
Request → Auth → Rate Limit → Transform → Route → Backend
```

**Design Patterns:**
- Gateway Pattern
- Middleware Pattern (chain of responsibility)
- Reverse Proxy Pattern

**Learning Outcomes:**
- Gateway responsibilities
- Middleware composition
- Cross-cutting concerns
- Protocol translation

---

### 50-Event-Store (Event Sourcing)
**Complexity:** Advanced

**Architecture Overview:**
- **Pattern:** Store state changes as events, not current state
- **Event Log:** Append-only, immutable history
- **Projection:** Current state computed from events
- **Benefits:** Audit trail, temporal queries, easy debugging

**Key Components:**
1. **Event Structure:**
   - Event ID (UUID)
   - Aggregate ID (entity identifier)
   - Event Type (UserCreated, OrderPlaced)
   - Payload (event data)
   - Timestamp
   - Version (sequence number per aggregate)
2. **Event Store:**
   - Append-only log storage
   - Optimistic concurrency control (version check)
   - Index by aggregate ID
3. **Aggregates:**
   - Rebuild by replaying events
   - Apply method per event type
   - In-memory state reconstruction
4. **Projections (Read Models):**
   - Event handlers update read models
   - Materialized views for queries
   - Async or synchronous updates
5. **Snapshots:**
   - Periodic state snapshots for performance
   - Replay from snapshot + recent events

**CQRS Integration:**
- Commands → Validate → Emit events
- Events → Projections → Read models
- Queries → Read models

**Design Patterns:**
- Event Sourcing Pattern
- CQRS Pattern
- Projection Pattern

**Learning Outcomes:**
- Event sourcing fundamentals
- State reconstruction
- Projection patterns
- Audit logging

---

### 51-Notification-Service
**Complexity:** Intermediate

**Architecture Overview:**
- **Channels:** Push (FCM/APNS), Email (SMTP), SMS
- **Priority:** High/medium/low with different queues
- **Batching:** Group notifications for efficiency
- **Delivery:** At-least-once with deduplication

**Key Components:**
1. **Notification Types:**
   - Push (mobile apps)
   - Email (transactional/marketing)
   - SMS (high urgency)
   - Webhook (callbacks)
2. **Queue Architecture:**
   - Priority queues per channel
   - Retry queues with backoff
   - Dead letter queue for failures
3. **Channel Providers:**
   - **Push:** Firebase Cloud Messaging (FCM), Apple Push (APNS)
   - **Email:** SMTP, SendGrid, SES
   - **SMS:** Twilio, AWS SNS
4. **User Preferences:**
   - Channel preferences per notification type
   - Opt-in/opt-out management
   - Quiet hours
5. **Delivery Tracking:**
   - Sent, Delivered, Opened, Failed states
   - Callback handling from providers
   - Retry logic with exponential backoff

**Rate Limiting:**
- Per-user limits (don't spam)
- Per-provider limits (API quotas)
- Adaptive throttling

**Design Patterns:**
- Notification Pattern
- Priority Queue Pattern
- Retry Pattern

**Learning Outcomes:**
- Multi-channel delivery
- Provider integration
- Priority handling
- Retry and deduplication

---

### 52-File-Storage (S3-like)
**Complexity:** Advanced

**Architecture Overview:**
- **Model:** Object storage with buckets and keys
- **Features:** Multi-part upload, metadata, ACL, signed URLs
- **Storage:** Erasure coding or replication for durability
- **Scalability:** Horizontal scaling with consistent hashing

**Key Components:**
1. **Object Model:**
   - Bucket (namespace)
   - Key (object identifier)
   - Data (binary payload)
   - Metadata (content-type, custom headers)
   - Version ID (optional versioning)
2. **Storage Backend:**
   - Chunk files across storage nodes
   - Erasure coding ( Reed-Solomon 6+3 = tolerate 3 failures)
   - Or replication (3 copies)
3. **Upload:**
   - **Single-part:** Direct upload for small files
   - **Multi-part:** Chunk large files, parallel upload
   - Resume capability (upload parts, complete)
4. **Download:**
   - Range requests (partial content)
   - Streaming response
   - CDN integration
5. **Metadata Store:**
   - Object metadata index
   - Bucket configuration
   - Access control lists
6. **Features:**
   - **Signed URLs:** Time-limited access tokens
   - **Lifecycle:** Auto-transition to cheaper storage, expiration
   - **Versioning:** Keep multiple versions of objects
   - **CORS:** Cross-origin resource sharing

**Architecture:**
- **Metadata Service:** Handles bucket/object metadata
- **Storage Nodes:** Store actual data chunks
- **Gateway:** Handles API requests, authentication

**Design Patterns:**
- Object Storage Pattern
- Erasure Coding Pattern
- Multi-part Upload Pattern

**Learning Outcomes:**
- Object storage architecture
- Erasure coding vs replication
- Large file handling
- Distributed storage systems

---

## Implementation Guide

### Project Structure Template

Each topic should follow this structure:

```
XX-Topic-Name/
├── main.go          # Go implementation
├── main.ts          # TypeScript implementation
├── README.md        # Architecture notes, design decisions
├── benchmark_test.go # Performance benchmarks (optional)
└── example/         # Usage examples (optional)
```

### Testing Strategy

1. **Unit Tests:** Core logic in isolation
2. **Integration Tests:** Multiple components
3. **Concurrency Tests:** Race condition detection
4. **Property-Based Tests:** Invariants (optional)
5. **Benchmarks:** Performance characteristics

### Documentation Format

Each implementation should document:

1. **Overview:** What and why
2. **Architecture:** Components and data flow
3. **API:** Public interface
4. **Complexity:** Time and space analysis
5. **Trade-offs:** Design decisions and alternatives
6. **Usage Examples:** Common patterns

### Performance Targets

| Complexity Level | Lines of Code | Time to Implement | Focus |
|-----------------|---------------|-------------------|-------|
| Entry-level | 100-200 | 2-4 hours | Core logic |
| Intermediate | 200-400 | 4-8 hours | Edge cases, tests |
| Advanced | 400-800 | 8-16 hours | Optimization, benchmarks |
| Expert | 800+ | 16+ hours | Production-ready features |

---

## Learning Path Recommendations

### Path 1: Foundation First
Start with Phase 1 (01-12) → Phase 2 (13-20) → Pick from Phases 3-5

Best for: Solid fundamentals before complex systems

### Path 2: Project-Driven
Pick an end goal from Phase 5 → Learn prerequisites as needed

Example: URL Shortener (43) needs:
- ID Generator (12)
- Key-Value Store (26) or use existing DB
- Rate Limiter (9)
- Caching (LRU - 01)

### Path 3: Distributed Systems Focus
Phase 1 (01-04, 08-12) → Phase 4 (31-42) → Phase 5 (43-52)

Best for: Backend engineers targeting distributed systems roles

### Path 4: Storage Engine Focus
Phase 1 (01-07) → Phase 3 (21-30) → 22 (LSM-Tree), 26 (KV Store)

Best for: Database/internals engineering

---

## Resources & References

### Books
- "Designing Data-Intensive Applications" by Martin Kleppmann
- "Database Internals" by Alex Petrov
- "System Design Interview" by Alex Xu

### Papers
- Dynamo: Amazon's Highly Available Key-value Store
- Bigtable: A Distributed Storage System for Structured Data
- The Google File System
- MapReduce: Simplified Data Processing on Large Clusters
- Raft: In Search of an Understandable Consensus Algorithm

### Courses
- MIT 6.824: Distributed Systems
- CMU 15-445: Database Systems

### Tools for Testing
- **Go:** `go test -race`, `go bench`, `pprof`
- **TypeScript:** `jest`, `benchmark.js`, `clinic.js`

---

## Success Metrics

Track your progress:

- [ ] Implemented in both Go and TypeScript
- [ ] Comprehensive test coverage (>80%)
- [ ] Benchmarks showing performance characteristics
- [ ] Documentation explaining design decisions
- [ ] Edge cases handled (errors, concurrency, limits)
- [ ] Production-ready features (monitoring, configuration)

---

## Final Notes

1. **Start Simple:** Implement core functionality first, optimize later
2. **Test Early:** Write tests as you build, not after
3. **Benchmark:** Measure before optimizing
4. **Document:** Explain why, not just what
5. **Iterate:** First version doesn't need to be perfect
6. **Compare:** Note differences between Go and TypeScript implementations

Good luck! Build consistently, and you'll master system design through implementation.
