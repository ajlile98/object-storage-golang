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
