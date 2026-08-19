package catalog

import (
	"context"
	"time"
)

type ObjectMetadata struct {
	Bucket      string
	Key         string
	VersionID   string
	BlobID      string
	Size        int64
	ContentType string
	Checksum    string
	CreatedAt   time.Time
	State       ObjectState
}

type ObjectState string

const (
	ObjectStatePending  ObjectState = "pending"
	ObjectStateReady    ObjectState = "ready"
	ObjectStateDeleting ObjectState = "deleting"
	ObjectStateDeleted  ObjectState = "deleted"
)

type ObjectStore interface {
	CreateObject(ctx context.Context, object ObjectMetadata) (ObjectMetadata, error)
	CompleteObjectUpload(ctx context.Context, object ObjectMetadata) (ObjectMetadata, error)
	GetObject(ctx context.Context, bucket, key string) (ObjectMetadata, error)
	MarkObjectDeleted(ctx context.Context, bucket, key string) error
}
