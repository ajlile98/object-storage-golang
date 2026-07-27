package object

import (
	"context"
	"io"
	"object-storage-golang/storage"
)

type ObjectService struct {
	blobs storage.BlobStore
}

type Object struct {
	Metadata ObjectMetadata
	Body     io.ReadCloser
}

type UploadResult struct {
	Size     int64
	Checksum string
}

func NewObjectService(blobs storage.BlobStore) *ObjectService {
	return &ObjectService{blobs: blobs}
}

func (service *ObjectService) Upload(
	ctx context.Context,
	key string,
	body io.Reader,
) (UploadResult, error) {
	size, checksum, err := service.blobs.Put(ctx, key, body)
	if err != nil {
		return UploadResult{0, ""}, err
	}

	// Persist object metadata: key, size, checksum, state, etc.
	_ = size
	_ = checksum
	return UploadResult{Size: size, Checksum: checksum}, nil
}

func (service *ObjectService) Read(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	return service.blobs.Get(ctx, key)
}

func (service *ObjectService) Delete(
	ctx context.Context,
	key string,
) error {
	return service.blobs.Delete(ctx, key)
}
