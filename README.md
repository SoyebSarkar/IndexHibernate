<p align="center">
  <img src="assets/logo.png" width="140" alt="Hiberstack logo" />
</p>

<h1 align="center">Hiberstack</h1>

<p align="center">
  Automatic cold storage & memory lifecycle management for search indexes
</p>

Hiberstack is a lightweight sidecar service that sits in front of a search engine (starting with **Typesense**) and automatically **offloads inactive collections** from memory to cold storage (disk / S3), then **reloads them on demand** when traffic returns.

It is designed to solve a very specific but recurring operational problem:

> Search engines like Typesense are extremely fast because their indexes live in RAM — but when you have *many collections* with *bursty access patterns*, RAM becomes expensive and eventually blocks writes.

Hiberstack adds **elasticity** to in-memory search engines *without modifying their internals*.

---

## Why Hiberstack exists

Typesense (and similar engines) are optimized for:

* blazing-fast queries
* in-memory indexes
* always-hot datasets

This works extremely well for **single or few collections** that are accessed continuously.

However, problems appear when you have:

* hundreds or thousands of collections
* most of them idle most of the time
* limited RAM budgets
* write operations that stall once memory is exhausted

Typical examples:

* multi-tenant SaaS platforms
* research or project-based tools
* AI / RAG systems with per-project indexes
* internal tools with bursty usage

Today, teams handle this manually using:

* cron jobs
* ad-hoc scripts
* manual cleanup
* over-provisioned RAM

Hiberstack makes this **automatic, safe, and observable**.

---

## What Hiberstack does

At a high level:

1. Tracks **collection-level activity** (queries & writes)
2. Identifies **inactive (cold) collections** based on policy (e.g. 6h idle)
3. Exports those collections to **cold storage** (JSONL + schema)
4. Deletes them from the search engine to **free RAM**
5. Transparently **reloads them on demand** when traffic returns

All of this happens **outside** the search engine, as a sidecar.

---

## What Hiberstack does NOT try to do

This project is intentionally opinionated.

Hiberstack does **NOT**:

* optimize single, always-hot large collections
* replace or fork search engines
* page data at document or segment level
* act as a general cache or LRU store

If you have:

* one large collection
* accessed continuously

Hiberstack is **not** for you.

---

## Who this project is for

### ✅ Good fit

Hiberstack is designed for teams that have:

* many collections (per project / tenant / workspace)
* bursty or sporadic access patterns
* RAM-constrained environments
* need for predictable memory usage

Examples:

* SaaS with per-customer indexes
* research platforms with per-study datasets
* AI tools creating indexes per workflow
* internal platforms running on limited infra

### ❌ Not a good fit

* single-collection deployments
* always-hot datasets
* latency-sensitive systems that cannot tolerate cold-starts

---

## Supported engines

Current support:

* **Typesense** (v0.x)

Planned (via adapters):

* Meilisearch
* OpenSearch / Elasticsearch

Hiberstack is **engine-agnostic by design**.

---

## Architecture overview

Hiberstack runs as a **standalone sidecar service**.

```
Client
  │
  ▼
Hiberstack (proxy + control plane)
  │
  ▼
Typesense (unmodified)
```

Cold storage:

* Local filesystem (default)
* S3-compatible object storage

---

## Operating modes

### Proxy mode (default)

```
Client → Hiberstack → Typesense
```

* All requests pass through Hiberstack
* Enables precise activity tracking
* Allows transparent reload-on-demand
* Adds sub-millisecond latency on hot paths

### Observer mode (future)

* Hiberstack does not proxy traffic
* Activity inferred from engine metrics
* No latency impact
* Limited reload automation

---

## Collection lifecycle

Each collection is managed via a simple state machine:

```
HOT      → loaded in memory
COLD     → offloaded to storage
LOADING  → reload in progress
FAILED   → last operation failed
```

Only one transition is allowed at a time per collection.

---

## Offload flow (background only)

Offloading **never happens on the request path**.

1. Background scheduler scans collections
2. Idle collections exceeding policy threshold are selected
3. Collection is exported (schema + documents)
4. Snapshot is stored safely
5. Collection is deleted from the engine
6. State transitions to `COLD`

This immediately frees RAM.

---

## Reload (on-demand) flow

Reload is triggered by access to a cold collection.

### Non-blocking reload (recommended)

```
Client → request
Proxy → state=COLD → trigger reload
Proxy → 202 WARMING_UP
```

Client retries after reload completes.

### Blocking reload (optional)

```
Client → request
Proxy → reload → forward request
```

Simpler, but introduces cold-start latency.

---

## Snapshot format

Snapshots are intentionally simple and portable:

```
snapshots/
  collection_name/
    schema.json
    documents.jsonl.gz
    metadata.json
```

This keeps recovery, debugging, and portability easy.

---

## Configuration example

```yaml
engine:
  type: typesense
  url: http://typesense:8108

mode: proxy

offload:
  after: 6h

reload:
  strategy: async

storage:
  type: s3
  bucket: index-hibernate
```

---

## Safety guarantees

Hiberstack is designed to be conservative:

* Collections are **never deleted** unless snapshot upload succeeds
* All operations are **idempotent**
* Per-collection locks prevent races
* Failures move collections to `FAILED` state

No silent data loss.

---

## Observability

Hiberstack exposes Prometheus metrics:

* `Hiberstack_collections_hot`
* `Hiberstack_collections_cold`
* `Hiberstack_offloads_total`
* `Hiberstack_reloads_total`
* `Hiberstack_reload_duration_seconds`

---

## Why a sidecar (and not engine internals)

* No fork or patching required
* Safe Typesense upgrades
* Clear separation of concerns
* Works across engines
* Easier to reason about failures

Hiberstack manages **lifecycle**, not search logic.

---

## Project status

* 🚧 Early-stage (v0.x)
* API may change
* Focused on correctness and safety first

---

## Roadmap

* [ ] Typesense adapter (v0)
* [ ] Local + S3 storage
* [ ] Async reloads
* [ ] Prometheus metrics
* [ ] Meilisearch adapter
* [ ] Pre-warming policies

---

## License

Apache 2.0

---

## Philosophy

Hiberstack is intentionally boring.

No magic. No heuristics. No clever tricks.

Just predictable, explicit control over memory —
for teams who need it.