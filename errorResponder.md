Updated todo list

- [ ] Define an `ErrorResponder` function type for HTTP error rendering.
- [ ] Change `rejectUnsafePaths` to accept an `ErrorResponder` and call it instead of `http.Error`.
- [ ] Add `writeS3HTTPError` in s3_api.go, which delegates to `writeS3Error` to produce S3-style XML.
- [ ] Wrap the router with the configured middleware in main.go:

  ```go
  handler := rejectUnsafePaths(writeS3HTTPError)(mux)
  ```

- [ ] Pass `handler` to `http.ListenAndServe`.
- [ ] Add `writeS3StorageError` to centralize mapping `ErrInvalidKey`, `ErrKeyConflict`, and unexpected errors into S3 HTTP/XML responses.
- [ ] Update `s3PutObjectHandler` to call `writeS3StorageError` instead of its local `switch`.
- [ ] Run `gofmt -w main.go s3_api.go` and `go test GitHub.`.

Keep the unsafe-path middleware outside `ServeMux`; it must run before route matching and path cleanup.