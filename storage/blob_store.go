package storage

import (
	"context"
	"errors"
	"io"
)

type BlobStore interface {
	Put(ctx context.Context, blobID string, data io.Reader) (size int64, checksum string, err error)
	Get(ctx context.Context, blobID string) (io.ReadCloser, error)
	Delete(ctx context.Context, blobID string) error
}

var ErrBlobNotFound = errors.New("blob not found")
