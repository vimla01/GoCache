# GoCache

A Redis-inspired in-memory key-value store built from scratch in Go, supporting concurrent clients, TTL-based expiration, and disk persistence over a custom TCP protocol.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Status](https://img.shields.io/badge/status-in%20development-yellow)]()

## Overview

**GoCache** is a from-scratch implementation of a Redis-like in-memory data store, built to understand how production key-value systems handle concurrency, networking, and durability under the hood. Rather than wrapping an existing library, this project implements its own TCP wire protocol, a concurrent-safe in-memory store, and a persistence/recovery layer - the same core building blocks real systems like Redis and Memcached rely on.

## Features

- **Core Commands** - `SET`, `GET`, `DELETE` with O(1) average-case lookups via an in-memory hash map.
- **TTL Expiration** - Keys can be set with a time-to-live; expired keys are evicted automatically via a background reaper, not just lazily on access.
- **Concurrent Client Handling** - Each client connection is handled on its own goroutine, with synchronization primitives (mutexes/RWMutex) protecting shared store state from race conditions under concurrent load.
- **Disk Persistence & Recovery** - Store state is periodically flushed to disk and reloaded on startup, so data survives a server restart (similar in spirit to Redis's RDB snapshotting).
- **Custom TCP Protocol** - A minimal, purpose-built request/response protocol over raw TCP sockets, eliminating HTTP overhead for low-latency client-server communication.

## Architecture

```
                    Clients
        ┌───────┐  ┌───────┐  ┌───────┐
        │Client1│  │Client2│  │Client3│
        └───┬───┘  └───┬───┘  └───┬───┘
            │ TCP      │ TCP      │ TCP
            ▼          ▼          ▼
     ┌─────────────────────────────────┐
     │         GoCache Server          │
     │  ┌────────────────────────────┐ │
     │  │   Connection Listener      │ │
     │  │  (goroutine per client)    │ │
     │  └─────────────┬──────────────┘ │
     │                ▼                │
     │  ┌────────────────────────────┐ │
     │  │  Command Parser/Protocol   │ │
     │  └─────────────┬──────────────┘ │
     │                ▼                │
     │  ┌────────────────────────────┐ │
     │  │  In-Memory Store (mutex-   │ │
     │  │  protected map + TTL heap) │ │
     │  └─────────────┬──────────────┘ │
     │                ▼                │
     │  ┌────────────────────────────┐ │
     │  │  Persistence Layer         │ │
     │  │  (snapshot to disk)        │ │
     │  └────────────────────────────┘ │
     └─────────────────────────────────┘
                     │
                     ▼
              [ disk storage ]
```

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go |
| Networking | Raw TCP sockets, custom protocol |
| Concurrency | Goroutines, Mutex/RWMutex |
| Persistence | Custom binary/snapshot format on disk |

## Supported Commands

| Command | Description |
|---|---|
| `SET key value [TTL]` | Stores a key-value pair, optionally with an expiration in seconds |
| `GET key` | Retrieves the value for a key, or a not-found response |
| `DELETE key` | Removes a key from the store |
| `PING` | Health check / connection liveness |

## Getting Started

### Prerequisites
- Go 1.21+

### Setup

```bash
# Clone the repository
git clone https://github.com/<your-username>/gocache.git
cd gocache

# Build the server
go build -o bin/gocache ./cmd/server

# Run the server (default port 6380)
./bin/gocache --port 6380
```

### Example Client Session

```bash
$ nc localhost 6380
SET name "vimla"
OK
GET name
"vimla"
SET session_token "abc123" 60
OK
DELETE name
OK
```

## Design Notes

- **Concurrency safety**: the store uses a single `RWMutex` guarding the underlying map, allowing concurrent reads while serializing writes - a deliberate trade-off favoring simplicity over sharded-lock complexity at this stage.
- **TTL expiration**: handled via a background goroutine that periodically sweeps for expired keys, avoiding unbounded memory growth from stale entries.
- **Persistence**: snapshots the in-memory state to disk at a configurable interval and on graceful shutdown, replaying the snapshot on startup to restore prior state.
- **Protocol**: a simple length-prefixed or delimiter-based text protocol (not full RESP) - designed to be easy to reason about and extend.

## Roadmap

- [ ] Sharded locking for higher write concurrency
- [ ] Append-only log (AOF) persistence alongside snapshotting
- [ ] LRU/LFU eviction policies under memory pressure
- [ ] Pub/Sub support
- [ ] Benchmark suite vs. Redis for throughput/latency comparison

## Why This Project

Key-value stores look simple on the surface but encode hard distributed-systems lessons: concurrency control, durability guarantees, and protocol design. GoCache was built to implement these mechanisms directly rather than treating Redis as a black box - focusing especially on Go's concurrency primitives and how they map to real-world server design.

