package storage

import (
	"context"
	"io"
)

type BlobStore interface {
	Put(ctx context.Context, key string, data io.Reader) (size int64, checksum string, err error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
