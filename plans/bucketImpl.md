You're right that the database topology is settled: **one SQLite file, two tables (`objects`, `buckets`)**. That's not the question. The question is purely at the **Go type level** — how many struct types wrap that shared `*sql.DB` and hang persistence methods off it. Those two decisions are orthogonal.

Let me lay out the two shapes.

**Option A — one store struct, methods for both tables**

```go
type SQLiteStore struct {
	db *sql.DB
}

func (s *SQLiteStore) CreateObject(ctx context.Context, o ObjectMetadata) (ObjectMetadata, error) { ... }
func (s *SQLiteStore) GetObject(ctx context.Context, bucket, key string) (ObjectMetadata, error) { ... }
func (s *SQLiteStore) CreateBucket(ctx context.Context, b BucketMetadata) error { ... }
func (s *SQLiteStore) BucketExists(ctx context.Context, name string) (bool, error) { ... }
```

**Option B — two store structs, each holding the same `*sql.DB`**

```go
type SQLiteObjectStore struct { db *sql.DB }
type SQLiteBucketStore struct { db *sql.DB }

// both constructed from the one *sql.DB you opened in main
obj := NewSQLiteObjectStore(db)
buc := NewSQLiteBucketStore(db)
```

Both options use the same DB and the same two tables. The difference is only how the Go code is partitioned.

**The deciding factor: cross-table transactions**

Here's the technical crux that makes this more than cosmetic. Some correct S3 operations must touch both tables *atomically*:

- `PUT object` — should insert into `objects` **only if** the bucket exists, ideally in one transaction so a concurrent `DELETE bucket` can't slip in between the check and the insert.
- `DELETE bucket` — should delete the bucket **only if** `objects` has no rows for it, again atomically.

A SQL transaction is a `*sql.Tx` obtained from **one** `*sql.DB` via `db.BeginTx(ctx)`. Every statement in that transaction must run on that same `*sql.Tx`.

- With **Option A**, this is natural: `BeginTx` on `s.db`, then run both the bucket check and the object insert on that one `tx`. One type owns both tables, so it can coordinate them in a single method.
- With **Option B**, it gets awkward. `SQLiteObjectStore` and `SQLiteBucketStore` each own their own methods but a transaction has to span both. You end up either (a) passing a `*sql.Tx` between the two structs, or (b) having one store reach into the other, or (c) introducing a third "transaction coordinator." All of that is coupling reintroduced to undo a split you didn't need — the exact "abstraction wearing a trench coat" problem.

So even though they share the `*sql.DB`, sharing the *handle* isn't enough; correct atomic operations want the *methods* to live on one type. That's the strongest argument for **Option A**.

**The idiomatic best-of-both: split the interfaces, not the concrete type**

Here's the Go-specific move that resolves the tension. In Go, interfaces are defined on the **consumer** side and satisfied implicitly. So you can keep **one** concrete `SQLiteStore` but let each service depend on a **narrow interface** describing only what it needs:

```go
// in catalog, defined where they're consumed
type ObjectStore interface {
	CreateObject(ctx context.Context, o ObjectMetadata) (ObjectMetadata, error)
	GetObject(ctx context.Context, bucket, key string) (ObjectMetadata, error)
	// ...
}

type BucketStore interface {
	CreateBucket(ctx context.Context, b BucketMetadata) error
	BucketExists(ctx context.Context, name string) (bool, error)
	// ...
}
```

One `*SQLiteStore` satisfies **both** interfaces automatically. You get:

- Interface segregation and easy mocking per service (a bucket-service test only needs a fake `BucketStore`).
- A single concrete type that can still run cross-table transactions internally.
- One `Initialize` that creates both tables, one constructor, one `*sql.DB` lifecycle.

**How this connects to "one service or two"**

The service-layer choice is separate and less constrained, because services don't hold the DB handle — they hold store interfaces. Reasonable options:

- **Two services** (`ObjectService`, `BucketService`), each taking the narrow interface it needs. Cleaner responsibilities; matches your `s3api` handler split (object handlers vs bucket handlers). I'd lean this way.
- **One service** with all methods. Simpler wiring, but it'll accumulate unrelated methods and blur the boundary.

Note: an operation like `PUT object` that needs both a bucket check and an object write can live on `ObjectService` and just depend on *both* `BucketStore` and `ObjectStore` interfaces — or, if it needs the atomic transaction, call a single `SQLiteStore` method that does the whole thing in one `tx`. That's another subtle nudge toward keeping the transactional logic in the store, not smeared across services.

**Recommendation**

- **Store:** one concrete `SQLiteStore` (Option A), exposed through two narrow interfaces `ObjectStore` and `BucketStore`.
- **Service:** two services, `ObjectService` and `BucketService`, each depending on the interface(s) it needs.
- **Schema:** exactly what you said — one DB file, `objects` and `buckets` tables, both created in `Initialize`.

Want me to start with the `object` → `catalog` rename + de-stutter, then scaffold the `BucketStore` interface, `buckets` table, and `BucketService` on top of the unified `SQLiteStore`?


You've correctly sniffed out that the concrete `SQLiteStore` gains nothing from the split — it satisfies whatever interfaces exist and is completely indifferent. So let me be honest about where the value actually is, and where I may have oversold it.

**The three possible justifications, ranked by how real they are here**

1. **Testing / mock surface (real, today).** This is the concrete win, and it's not nothing. If `BucketService` depends on a narrow `BucketStore`, a `BucketService` test fakes only the two or three bucket methods. If it depends on a fat combined interface, every test double has to stub out all the object methods too, even the ones that method never calls. Narrow interfaces = smaller fakes = less test churn when unrelated methods change. That benefit exists with a single implementation.

2. **Contract clarity (real, minor).** `NewBucketService(store BucketStore)` documents in the type signature exactly what the service touches. A reader (or you, in six months) sees the dependency surface without opening the implementation. Mild but genuine.

3. **Swappable implementations (speculative — you're right to be skeptical).** "Someday a Postgres or in-memory store" is the classic over-engineering trap. You have one implementation and no committed second one. Designing the split *for* that reason would be YAGNI. So don't count this one.

So: **it's mostly #1 with a bit of #2. It is not justified by multiple implementations.**

**The honest counter-argument (Go idiom actually leans your way)**

There's a well-known Go proverb: *"Don't design with interfaces, discover them."* Interfaces are cheap precisely because they're consumer-side and implicit, which means you can add them exactly when a consumer needs them — you don't lose anything by waiting. You already have a `MetadataStore` interface, so you've "discovered" one; the question is only whether to split it, and that split can also be deferred until a test or service actually feels the friction.

There's also a real downside to over-splitting that I flagged earlier: the atomic `PUT object` (check bucket + insert object in one transaction) doesn't decompose cleanly across two narrow interfaces — it wants to be a single method on the store. Split too eagerly and you'll find operations that straddle the boundary and force you to either widen an interface or add a third one.

**Pragmatic recommendation**

Given it's a learning project and you already use the interface pattern:

- **Define each interface at the consumer, only as wide as that consumer uses.** When you write `BucketService`, define `BucketStore` next to it with just the bucket methods it calls. That's *discovering* the interface, not speculating — fully idiomatic. Same for the object side.
- **Keep one concrete `SQLiteStore`** satisfying both, so cross-table transactions stay easy.
- **Don't create the interfaces before the services that consume them.** If you're scaffolding bottom-up and haven't written `BucketService` yet, it's fine to start with plain methods on `SQLiteStore` and introduce the interface the moment the service (or its test) exists.

In other words: the split is worth doing for the testing/contract payoff, but let it be *pulled into existence by a consumer that needs it*, rather than declared upfront as architecture. That keeps you on the right side of both the "narrow interfaces" and the "don't speculate" guidance.

Practically, for our build order: I'd write `BucketService` + its test first, and let that test be the thing that motivates the `BucketStore` interface. Want to go that route?


One SQLite gotcha worth knowing for this exact pattern: db.BeginTx starts a deferred transaction, which takes only a read lock at your SELECT and tries to upgrade to a write lock at the INSERT. If another connection wrote in between, that upgrade can fail with SQLITE_BUSY_SNAPSHOT, and you must retry the whole transaction. The clean fix is to start the transaction as a writer from the beginning (BEGIN IMMEDIATE), so it grabs the write lock before the SELECT and no one can change the bucket underneath you. Practically, in Go you'd either issue BEGIN IMMEDIATE or rely on the busy-timeout pragma you already set (busy_timeout(5000)) plus a retry loop.