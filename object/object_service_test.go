package object

import (
	"context"
	"io"
	"object-storage-golang/storage"
	"strings"
	"testing"
)

func newTestBlobStore(t *testing.T) storage.BlobStore {
	t.Helper()

	blobStore := &storage.FileSystemBlobStore{
		Root: t.TempDir(),
	}

	return blobStore
}
func TestObjectUpload(t *testing.T) {
	ctx := context.Background()
	blobStore := newTestBlobStore(t)
	metadataStore := newTestMetadataStore(t)
	objectService := NewObjectService(blobStore, metadataStore)

	_, err := objectService.Upload(
		ctx,
		"photos",
		"summer.jpg",
		"image/jpg",
		strings.NewReader("image bytes"),
	)
	if err != nil {
		t.Fatal(err)
	}

	object, err := objectService.Read(ctx, "photos", "summer.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer object.Body.Close()

	got, err := io.ReadAll(object.Body)
	if err != nil {
		t.Fatal(err)
	}

	want := []byte("image bytes")
	if string(got) != string(want) {
		t.Fatalf("object = %q, want %q", got, want)
	}

}
