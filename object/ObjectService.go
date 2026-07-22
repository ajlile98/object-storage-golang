package object

import (
	"context"
	"io"
	"object-storage-golang/storage"
)

type ObjectService struct {
	blobs storage.BlobStore
}

// Eventually this service can revolve around Objects,
// once metadata service is involved this makes more sense
type Object struct {
	Body        io.ReadCloser
	Size        int64
	ContentType string
	Checksum    string
}

type UploadResult struct {
	Size int64
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
