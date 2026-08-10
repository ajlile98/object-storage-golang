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
	mux.HandleFunc("DELETE /{bucket}/{key...}", S3DeleteObjectHandler(service))

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

	putETag := putResponse.Header.Get("ETag"); 
	if putETag == "" {
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

	if contentType := getResponse.Header.Get("Content-Type"); contentType != "image/jpeg" {
		t.Fatalf("Content-Type = %q, want %q", contentType, "image/jpeg")
	}

	if ETag := getResponse.Header.Get("ETag"); ETag != putETag {
		t.Fatalf("ETag = %q, want %q", ETag, putETag)
	}

	if length := getResponse.Header.Get("Content-Length"); length != "11" {
		t.Fatalf("Length = %q, want %q", length, "11")
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

func TestDeleteLifecycle(t *testing.T) {
	mux := newTestMux(t)

	putRequest := httptest.NewRequest(
		"PUT",
		"/photos/summer.jpg",
		strings.NewReader("image bytes"),
	)

	putRecorder := httptest.NewRecorder()

	mux.ServeHTTP(putRecorder, putRequest)

	putResponse := putRecorder.Result()
	defer putResponse.Body.Close()

	if putResponse.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", putResponse.StatusCode, http.StatusOK)
	}

	deleteRequest := httptest.NewRequest(
		"DELETE",
		"/photos/summer.jpg",
		nil,
	)

	deleteRecorder := httptest.NewRecorder()

	mux.ServeHTTP(deleteRecorder, deleteRequest)

	deleteResponse := deleteRecorder.Result()
	defer deleteResponse.Body.Close()

	if deleteResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", deleteResponse.StatusCode, http.StatusNoContent)
	}

	getRequest := httptest.NewRequest(
		"GET",
		"/photos/summer.jpg",
		nil,
	)

	getRecorder := httptest.NewRecorder()

	mux.ServeHTTP(getRecorder, getRequest)

	getResponse := getRecorder.Result()
	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", getResponse.StatusCode, http.StatusNoContent)
	}
}

func TestGetAfterDelete(t *testing.T) {
	mux := newTestMux(t)

	putRequest := httptest.NewRequest(
		"PUT",
		"/photos/summer.jpg",
		strings.NewReader("first image bytes"),
	)

	putRecorder := httptest.NewRecorder()

	mux.ServeHTTP(putRecorder, putRequest)

	putResponse := putRecorder.Result()
	defer putResponse.Body.Close()

	if putResponse.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", putResponse.StatusCode, http.StatusOK)
	}

	putRequest2 := httptest.NewRequest(
		"PUT",
		"/photos/summer.jpg",
		strings.NewReader("second image bytes"),
	)

	putRecorder2 := httptest.NewRecorder()

	mux.ServeHTTP(putRecorder2, putRequest2)

	putResponse2 := putRecorder2.Result()
	defer putResponse2.Body.Close()

	if putResponse2.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", putResponse2.StatusCode, http.StatusOK)
	}

	deleteRequest := httptest.NewRequest(
		"DELETE",
		"/photos/summer.jpg",
		nil,
	)

	deleteRecorder := httptest.NewRecorder()

	mux.ServeHTTP(deleteRecorder, deleteRequest)

	deleteResponse := deleteRecorder.Result()
	defer deleteResponse.Body.Close()

	if deleteResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", putResponse.StatusCode, http.StatusNoContent)
	}

	getRequest := httptest.NewRequest(
		"GET",
		"/photos/summer.jpg",
		nil,
	)

	getRecorder := httptest.NewRecorder()

	mux.ServeHTTP(getRecorder, getRequest)

	getResponse := getRecorder.Result()
	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", getResponse.StatusCode, http.StatusNoContent)
	}
}
