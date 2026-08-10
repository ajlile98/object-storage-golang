Updated todo list

GET, PUT, and DELETE object operations are the right core, but they are only the first layer of S3 compatibility. “Fully S3-compatible” is a large target; the practical goal is a documented subset that works with selected clients, then expand it deliberately.

**Essential Next Steps**

1. **Request authentication**
   - AWS Signature Version 4 (`Authorization`, canonical request, signing key derivation).
   - Access keys, secret storage, clock-skew errors, and request expiry.
   - HTTPS in real deployments.

2. **Bucket operations**
   - `PUT /{bucket}`: create bucket.
   - `DELETE /{bucket}`: delete an empty bucket.
   - `HEAD /{bucket}`: verify/access bucket.
   - `GET /`: list buckets.
   - Bucket existence and ownership should live in metadata, not be inferred from filesystem directories.

3. **Object operations expected by clients**
   - `HEAD /{bucket}/{key}` for metadata without the body.
   - `GET /{bucket}?list-type=2...` for `ListObjectsV2`.
   - `CopyObject` via `x-amz-copy-source`.
   - Conditional requests: `If-Match`, `If-None-Match`, and related behavior.
   - Object metadata headers, especially `Content-Type`, `Content-Length`, `ETag`, and `x-amz-meta-*`.

4. **HTTP object semantics**
   - Byte ranges: `Range: bytes=...` and `206 Partial Content`.
   - Correct `Content-Length`.
   - MIME type persistence and replay.
   - Proper handling of client cancellation during streaming.
   - XML error responses with S3 error codes, request IDs, and resource fields.

5. **Multipart uploads**
   - Initiate: `POST /{bucket}/{key}?uploads`
   - Upload part: `PUT /{bucket}/{key}?partNumber=N&uploadId=...`
   - Complete: `POST /{bucket}/{key}?uploadId=...`
   - Abort: `DELETE /{bucket}/{key}?uploadId=...`
   - List parts/uploads.
   - This is necessary for many large-file clients and is a substantial feature on its own.

**Metadata You Need**

Your filesystem backend currently has the bytes and can calculate a checksum, but an S3 API needs durable metadata:

```text
Bucket
Object key
Version / generation
Size
ETag
Content-Type
User metadata
Creation / update time
Storage location or blob ID
Object state
```

That is the point where a metadata database becomes valuable. Keep the filesystem backend responsible for byte storage, while an object/bucket service owns this metadata and coordinates writes.

**Compatibility Details**

A mature implementation eventually needs:

- Versioning and delete markers
- Bucket policies / IAM-style authorization
- ACLs, if you choose to support them
- Server-side encryption
- Lifecycle policies
- Object tags
- CORS
- Presigned URLs
- Event notifications
- Object locking / retention policies
- Storage classes and restore workflows

You do not need these to have a useful object store.

**Recommended Milestones**

```text
Milestone 1: Anonymous S3-style object store
- PUT, GET, HEAD, DELETE object
- Buckets
- ListObjectsV2
- XML errors
- Metadata and range GETs

Milestone 2: Compatible client authentication
- SigV4
- Access keys
- Presigned GET/PUT URLs

Milestone 3: Practical large-object support
- Multipart upload
- CopyObject
- Conditional requests
- Durable metadata

Milestone 4: Advanced S3 features
- Versioning, policies, lifecycle, tags, encryption
```

The best definition of compatibility is behavioral: choose a client such as the AWS Go SDK, AWS CLI, or `rclone`, write integration tests for the operations you claim to support, and describe the project as an **S3-compatible subset** until those tests demonstrate broader coverage.