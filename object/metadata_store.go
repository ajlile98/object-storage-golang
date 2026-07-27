package object

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

type MetadataStore interface {
	Create(ctx context.Context, object ObjectMetadata) (ObjectMetadata, error)
	MarkReady(ctx context.Context, bucket, key, version string) error
	Get(ctx context.Context, bucket, key string) (ObjectMetadata, error)
	// GetVersion(ctx context.Context, bucket, key, versionID string) (ObjectMetadata, error)
	MarkDeleted(ctx context.Context, bucket, key, version_id string) error
}
