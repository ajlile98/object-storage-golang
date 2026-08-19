package catalog

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestMetadataStore(t *testing.T) *SQLiteStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "metadata.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	store := NewSQLiteStore(db)

	if err := store.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}

	return store
}

func TestSQLiteMetadataStoreCreate(t *testing.T) {
	store := newTestMetadataStore(t)
	ctx := context.Background()

	created, err := store.CreateObject(ctx, ObjectMetadata{
		Bucket:      "photos",
		Key:         "summer.jpg",
		VersionID:   "version-1",
		BlobID:      "blob-1",
		ContentType: "image/jpeg",
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.Bucket != "photos" {
		t.Fatalf("bucket = %q, want %q", created.Bucket, "photos")
	}

	if created.Key != "summer.jpg" {
		t.Fatalf("key = %q, want %q", created.Key, "summer.jpg")
	}

	if created.VersionID != "version-1" {
		t.Fatalf("version ID = %q, want %q", created.VersionID, "version-1")
	}

	if created.BlobID != "blob-1" {
		t.Fatalf("blob ID = %q, want %q", created.BlobID, "blob-1")
	}

	if created.ContentType != "image/jpeg" {
		t.Fatalf("content type = %q, want %q", created.ContentType, "image/jpeg")
	}

	if created.State != ObjectStatePending {
		t.Fatalf("state = %q, want %q", created.State, ObjectStatePending)
	}

	if created.CreatedAt.IsZero() {
		t.Fatal("created at is zero, want database timestamp")
	}
}

func TestMetadataCompleteUpload(t *testing.T) {
	store := newTestMetadataStore(t)
	ctx := context.Background()

	created, err := store.CreateObject(ctx, ObjectMetadata{
		Bucket:      "photos",
		Key:         "summer.jpg",
		VersionID:   "version-1",
		BlobID:      "blob-1",
		ContentType: "image/jpeg",
	})
	if err != nil {
		t.Fatal(err)
	}

	created.Size = 42
	created.Checksum = "checksum-1"

	completed, err := store.CompleteObjectUpload(ctx, created)
	if err != nil {
		t.Fatal(err)
	}

	found, err := store.GetObject(ctx, "photos", "summer.jpg")
	if err != nil {
		t.Fatal(err)
	}

	if found.Bucket != completed.Bucket {
		t.Fatalf("bucket = %q, want %q", found.Bucket, completed.Bucket)
	}

	if found.Key != completed.Key {
		t.Fatalf("key = %q, want %q", found.Key, completed.Key)
	}

	if found.VersionID != completed.VersionID {
		t.Fatalf("version ID = %q, want %q", found.VersionID, completed.VersionID)
	}

	if found.BlobID != completed.BlobID {
		t.Fatalf("blob ID = %q, want %q", found.BlobID, completed.BlobID)
	}

	if found.Size != completed.Size {
		t.Fatalf("size = %d, want %d", found.Size, completed.Size)
	}

	if found.Checksum != completed.Checksum {
		t.Fatalf("checksum = %q, want %q", found.Checksum, completed.Checksum)
	}

	if found.State != ObjectStateReady {
		t.Fatalf("state = %q, want %q", found.State, ObjectStateReady)
	}

}

func TestMetadataPendingUploadIsInvisible(t *testing.T) {
	store := newTestMetadataStore(t)
	ctx := context.Background()

	_, err := store.CreateObject(ctx, ObjectMetadata{
		Bucket:      "photos",
		Key:         "summer.jpg",
		VersionID:   "version-1",
		BlobID:      "blob-1",
		ContentType: "image/jpeg",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.GetObject(ctx, "photos", "summer.jpg")
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatal(err)
	}

}
