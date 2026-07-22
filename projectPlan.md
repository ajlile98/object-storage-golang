Updated todo list

Yes. An HTTP server is a sensible interface for the data-storage layer, especially if your object storage may eventually support remote clients or multiple storage nodes. But it is usually best to treat HTTP as a **transport adapter**, not as the storage abstraction itself.

A useful split is:

```text
Client / API
    |
Object service (authentication, buckets, object operations)
    |
Metadata store (names, versions, sizes, checksums, locations, lifecycle state)
    |
Storage interface
    |
Storage backend(s): local disk, remote HTTP nodes, cloud provider, memory
```

In Go, the core service could depend on an interface like:

```go
type BlobStore interface {
    Put(ctx context.Context, key string, data io.Reader) (size int64, checksum string, err error)
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
}
```

That lets you begin with a filesystem implementation, while keeping an HTTP-backed implementation available later:

```text
FilesystemBlobStore  -> writes blobs to local disk
MemoryBlobStore      -> useful for tests
HTTPBlobStore        -> forwards blob operations to a storage node
S3BlobStore          -> uses S3-compatible storage as the data layer
```

**Architecture options**

1. **Single-process, local disk**
   - Metadata: SQLite, BoltDB, or Postgres.
   - Data: files on the server filesystem.
   - Best starting point for a personal project.
   - Keep metadata in the database and payload bytes in files; avoid storing large object data directly in a relational database initially.

2. **API server plus HTTP storage nodes**
   - A metadata/API service records where each object lives.
   - Dedicated storage nodes expose internal endpoints such as `PUT /blobs/{id}` and `GET /blobs/{id}`.
   - This is a natural step toward distribution, replication, and capacity expansion.
   - The HTTP storage API should be private/internal; clients talk only to the public object API.

3. **S3-compatible public API**
   - Implement an S3-like API surface for clients.
   - Store data locally or on your own HTTP nodes behind it.
   - More ambitious because authentication/signing, multipart uploads, range requests, and compatibility details add up quickly.

4. **Chunked/content-addressed storage**
   - Split objects into fixed or variable-sized chunks.
   - Store chunks by checksum, for example `sha256/<hash>`.
   - Metadata maps object/version to ordered chunk hashes.
   - Enables integrity checks, deduplication, replication, and large-object streaming. This is closer to how serious object stores are built, but adds complexity.

5. **Delegate the data layer**
   - Your project owns metadata, policies, versioning, and indexing.
   - Bytes live in S3, MinIO, Azure Blob, or another object store.
   - Good if your goal is the metadata/control-plane design rather than filesystem/distributed-storage mechanics.

**Metadata design**

Metadata should normally include:

- Bucket and object key
- Object version or generation number
- Content length and MIME type
- ETag/checksum
- Creation/update timestamps
- Storage location or chunk list
- Object state: `pending`, `ready`, `deleting`, `deleted`
- Optional user metadata and tags

The state is important because writing object bytes and committing metadata are not one database transaction when data is on disk or another HTTP service.

A practical write flow:

```text
1. Receive upload and stream it to temporary blob storage.
2. Calculate size and checksum while streaming.
3. Persist metadata as pending, or prepare its final record.
4. Atomically move/rename the temporary blob to its final location.
5. Mark the object version ready in metadata.
```

For deletion:

```text
1. Mark the object/version as deleting or deleted in metadata.
2. Remove its underlying bytes asynchronously.
3. Reconcile orphaned blobs and missing blobs with a background job.
```

That reconciliation process is worth building even in a small project. It handles crashes between metadata updates and physical writes.

**Recommendation for your first iteration**

Start with:

- An HTTP API service for clients.
- SQLite or Postgres for metadata.
- A `BlobStore` interface.
- A filesystem-backed blob store that writes each object to a deterministic path.
- Streaming reads/writes using `io.Reader` and `io.ReadCloser`.
- Checksums, object states, and atomic temporary-file-to-final-file moves.

Later, add an `HTTPBlobStore` implementation and move storage into separate nodes without changing the metadata service’s core behavior. That preserves a clean boundary: the metadata layer decides *what an object is and where it belongs*; the blob store is responsible only for *reliably storing and retrieving bytes*.