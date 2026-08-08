package s3api

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"object-storage-golang/object"
	"object-storage-golang/storage"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestMetadataStore(t *testing.T) *object.SQLiteMetadataStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "metadata.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	store := object.NewSQLiteMetadataStore(db)

	if err := store.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}

	return store
}

func newTestBlobStore(t *testing.T) storage.BlobStore {
	t.Helper()

	blobStore := &storage.FileSystemBlobStore{
		Root: t.TempDir(),
	}

	return blobStore
}

func newTestObjectService(t *testing.T) *object.ObjectService {
	bs := newTestBlobStore(t)
	ms := newTestMetadataStore(t)

	return object.NewObjectService(bs, ms)
}

func newTestMux(t *testing.T) *http.ServeMux {
	service := newTestObjectService(t)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /{bucket}/{key...}", S3PutObjectHandler(service))
	mux.HandleFunc("GET /{bucket}/{key...}", S3GetObjectHandler(service))

	return mux
}

func TestPutObjectThenGetObject(t *testing.T) {
	mux := newTestMux(t)

	putRequest := httptest.NewRequest(
		http.MethodPut,
		"/photos/summer.jpg",
		strings.NewReader("image bytes"),
	)
	putRequest.Header.Set("Content-Type", "image/jpeg")

	putRecorder := httptest.NewRecorder()

	mux.ServeHTTP(putRecorder, putRequest)

	putResponse := putRecorder.Result()
	defer putResponse.Body.Close()

	if putResponse.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", putResponse.StatusCode, http.StatusOK)
	}

	if etag := putResponse.Header.Get("ETag"); etag == "" {
		t.Fatal("ETag is empty")
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/photos/summer.jpg",
		nil,
	)

	getRecorder := httptest.NewRecorder()

	mux.ServeHTTP(getRecorder, getRequest)

	getResponse := getRecorder.Result()
	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", getResponse.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "image bytes" {
		t.Fatalf("body = %q, want %q", body, "image bytes")
	}

}

func TestLatestUploadWins(t *testing.T) {
	mux := newTestMux(t)

	firstRequest := httptest.NewRequest(
		http.MethodPut,
		"/photos/summer.jpg",
		strings.NewReader("first image bytes"),
	)
	firstRequest.Header.Set("Content-Type", "image/jpeg")

	secondRequest := httptest.NewRequest(
		http.MethodPut,
		"/photos/summer.jpg",
		strings.NewReader("second image bytes"),
	)
	secondRequest.Header.Set("Content-Type", "image/jpeg")

	firstRecorder := httptest.NewRecorder()
	secondRecorder := httptest.NewRecorder()

	mux.ServeHTTP(firstRecorder, firstRequest)
	mux.ServeHTTP(secondRecorder, secondRequest)

	firstResponse := firstRecorder.Result()
	defer firstResponse.Body.Close()
	secondResponse := secondRecorder.Result()
	defer secondResponse.Body.Close()

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/photos/summer.jpg",
		nil,
	)

	getRecorder := httptest.NewRecorder()

	mux.ServeHTTP(getRecorder, getRequest)

	getResponse := getRecorder.Result()
	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", getResponse.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "second image bytes" {
		t.Fatalf("body = %q, want %q", body, "second image bytes")
	}
}
