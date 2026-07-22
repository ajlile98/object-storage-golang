package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"object-storage-golang/storage"
	"object-storage-golang/object"
	"object-storage-golang/s3api"
	"strings"
)

func rejectUnsafePaths(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawPath := r.URL.EscapedPath()

		for _, rawSegment := range strings.Split(rawPath, "/") {
			segment, err := url.PathUnescape(rawSegment)
			if err != nil || segment == "." || segment == ".." ||
				strings.Contains(segment, "/") || strings.Contains(segment, `\`) {
				http.Error(w, "invalid object key", http.StatusBadRequest)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func uploadHandler(service *object.ObjectService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		size, checksum, err := service.Blobs.Put(r.Context(), r.PathValue("key"), r.Body)
		if err != nil {
			if errors.Is(err, storage.ErrKeyConflict) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			} else if errors.Is(err, storage.ErrInvalidKey) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else {
				http.Error(w, "failed to store object", http.StatusInternalServerError)
				return
			}
		}
		fmt.Fprintf(w, "\nstored %d bytes with checksum %s\n", size, checksum)
	}
}

func main() {
	// Create a new request multiplexer (router)
	blobStore := &storage.FileSystemBlobStore{
		Root: "./data",
	}
	service := object.NewObjectService(blobStore)
	mux := http.NewServeMux()

	// Register the handler function for the root path
	mux.HandleFunc("PUT /{bucket}/{key...}", s3api.S3PutObjectHandler(service))

	fmt.Println("Server starting on http://localhost:8080")

	// Start the server and listen on port 8080
	if err := http.ListenAndServe(":8080", rejectUnsafePaths(mux)); err != nil {
		panic(err)
	}
}
