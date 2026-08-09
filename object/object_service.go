package object

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"object-storage-golang/storage"
)

type ObjectService struct {
	blobs         storage.BlobStore
	metadataStore MetadataStore
}

type Object struct {
	Metadata ObjectMetadata
	Body     io.ReadCloser
}

type UploadResult struct {
	Metadata ObjectMetadata
}

func NewObjectService(
	blobs storage.BlobStore,
	metadataStore MetadataStore,
) *ObjectService {
	return &ObjectService{blobs: blobs, metadataStore: metadataStore}
}

func (service *ObjectService) Upload(
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

	pending, err := service.metadataStore.Create(ctx, ObjectMetadata{
		Bucket:      bucket,
		Key:         key,
		VersionID:   versionID,
		BlobID:      blobID,
		ContentType: contentType,
	})
	if err != nil {
		return ObjectMetadata{}, err
	}

	size, checksum, err := service.blobs.Put(ctx, blobID, body)
	if err != nil {
		return ObjectMetadata{}, err
	}

	pending.Size = size
	pending.Checksum = checksum

	return service.metadataStore.CompleteUpload(ctx, pending)
}

func (service *ObjectService) Read(
	ctx context.Context,
	bucket,
	key string,
) (Object, error) {
	objectMetadata, err := service.metadataStore.Get(ctx, bucket, key)
	if err != nil {
		return Object{}, fmt.Errorf("read object metadata %s/%s: %w", bucket, key, err)
	}
	
	objectData, err := service.blobs.Get(ctx, objectMetadata.BlobID)
	if err != nil {
		return Object{}, fmt.Errorf("read object %s/%s: %w", bucket, key, err)
	}
	return Object{
		Metadata: objectMetadata,
		Body: objectData,
	}, nil
}

func (service *ObjectService) Delete(
	ctx context.Context,
	bucket,
	key string,
) error {
	err := service.metadataStore.MarkDeleted(ctx, bucket, key, "")
	if err != nil {
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
