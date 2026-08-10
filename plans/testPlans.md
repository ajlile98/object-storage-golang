1. Metadata lookup after completed upload
- In sqlite_metadata_store_test.go.
- Create a pending object, complete it with size/checksum, then call Get(bucket, key).
- Assert it returns the ready metadata, especially its BlobID.
- This is the immediate dependency of the fix: Read needs metadata to resolve bucket/key to the generated blob ID.

2. Pending uploads are invisible
- Create metadata but do not call CompleteUpload.
- Get(bucket, key) should return ErrObjectNotFound.
- This protects against exposing incomplete or failed uploads when you change Read to use metadata.

3. Latest ready upload wins
- Complete two uploads for the same bucket/key with different versions and blob IDs.
- Get(bucket, key) should return the newest ready version.
- This locks in the current intended version-selection query before service code starts relying on it.

4. Service-level upload then read
- In object/object_service_test.go, with real temporary SQLite and filesystem storage.
- Call Upload, then Read(bucket, key) once the API is redesigned to accept bucket and key separately.
- Assert the bytes match.
- This is more focused than HTTP and pinpoints whether a failure is in service coordination or the handler.

1. Delete lifecycle — completely untested (highest priority)
Delete was the headline bugfix (it used to always 500), and nothing exercises it now:

# tests before feature branch close

Delete → Get returns NotFound/NoSuchKey (the core contract).
Delete of a missing key is idempotent — this directly covers the errors.Is(err, ErrObjectNotFound) swallow we added in Delete; a regression there would resurface as a 500.
MarkDeleted at the metadata layer: marks the latest ready version, and returns ErrObjectNotFound when there's nothing to delete.
s3api DELETE handler returns 204 (there's no DELETE route even wired into newTestMux).
2. Get-after-delete must not resurrect an older version
This is the exact subtle change we made to Get (return latest row regardless of state, reject if not ready). The scenario has no coverage: upload v1 → upload v2 → delete → Get must be NotFound, not fall back to v1. Easy to silently break if someone re-adds a state = ready filter.

3. The storage package has NO tests at all
We heavily reworked it (atomic temp+rename, prefix sharding, blobID keying, validateID) and there's not a single _test.go in storage. Worth adding:

validateID rejects empty, non-hex, wrong length, and path-ish inputs (../…) — this is a security boundary, so it matters most.
Put→Get round trip returns identical bytes, correct size and sha256 checksum.
Blob lands at the sharded path (Root/ab/cd/<id>).
Get of a missing blob → ErrObjectNotFound; Delete of a missing blob → nil (idempotent).
4. GET response headers we just added
TestPutObjectThenGetObject checks the body and the PUT ETag, but nothing asserts the GET now returns the stored Content-Type (not the old hardcoded application/octet-stream) or a correct Content-Length. Since serving real headers was a deliberate change, a test pins it.

Lower priority (not from this branch, but gaps)
RejectUnsafePaths middleware has no test despite being security-relevant (path-traversal rejection).
s3api error translations generally: invalid-key → 400.
The two I'd treat as must-have before merging this branch are #1 (delete lifecycle) and #2 (no resurrection) — they cover the exact bugs/behaviors this branch introduced. #3 is the biggest raw coverage hole.

