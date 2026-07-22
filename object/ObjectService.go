package object

import (
	"context"
	"io"
	"object-storage-golang/storage"
)

type ObjectService struct {
	Blobs storage.BlobStore
}

func NewObjectService(Blobs storage.BlobStore) *ObjectService {
	return &ObjectService{Blobs: Blobs}
}

func (service *ObjectService) Upload(
	ctx context.Context,
	key string,
	body io.Reader,
) error {
	size, checksum, err := service.Blobs.Put(ctx, key, body)
	if err != nil {
		return err
	}

	// Persist object metadata: key, size, checksum, state, etc.
	_ = size
	_ = checksum
	return nil
}
