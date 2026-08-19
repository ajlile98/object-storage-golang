package catalog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"object-storage-golang/storage"
)

type ObjectService struct {
	blobs         storage.BlobStore
	metadataStore ObjectStore
}

type Object struct {
	Metadata ObjectMetadata
	Body     io.ReadCloser
}

func NewObjectService(
	blobs storage.BlobStore,
	metadataStore ObjectStore,
) *ObjectService {
	return &ObjectService{blobs: blobs, metadataStore: metadataStore}
}

func (s *ObjectService) Upload(
	ctx context.Context,
	bucket string,
	key string,
	contentType string,
	body io.Reader,
) (ObjectMetadata, error) {

	versionID, err := newID()
	if err != nil {
		return ObjectMetadata{}, err
	}

	blobID, err := newID()
	if err != nil {
		return ObjectMetadata{}, err
	}

	pending, err := s.metadataStore.CreateObject(ctx, ObjectMetadata{
		Bucket:      bucket,
		Key:         key,
		VersionID:   versionID,
		BlobID:      blobID,
		ContentType: contentType,
	})
	if err != nil {
		return ObjectMetadata{}, err
	}

	size, checksum, err := s.blobs.Put(ctx, blobID, body)
	if err != nil {
		return ObjectMetadata{}, err
	}

	pending.Size = size
	pending.Checksum = checksum

	return s.metadataStore.CompleteObjectUpload(ctx, pending)
}

func (s *ObjectService) Read(
	ctx context.Context,
	bucket,
	key string,
) (Object, error) {
	objectMetadata, err := s.metadataStore.GetObject(ctx, bucket, key)
	if err != nil {
		return Object{}, fmt.Errorf("read object metadata %s/%s: %w", bucket, key, err)
	}

	objectData, err := s.blobs.Get(ctx, objectMetadata.BlobID)
	if err != nil {
		return Object{}, fmt.Errorf("read object %s/%s: %w", bucket, key, err)
	}
	return Object{
		Metadata: objectMetadata,
		Body:     objectData,
	}, nil
}

func (s *ObjectService) Delete(
	ctx context.Context,
	bucket,
	key string,
) error {
	err := s.metadataStore.MarkObjectDeleted(ctx, bucket, key)
	if err != nil && !errors.Is(err, ErrObjectNotFound) {
		return fmt.Errorf("mark object deleted %s/%s: %w", bucket, key, err)
	}
	return nil
}

func newID() (string, error) {
	bytes := make([]byte, 16) // 128 random bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate identifier: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}
