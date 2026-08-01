package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"object-storage-golang/httpmiddleware"
	"object-storage-golang/object"
	"object-storage-golang/s3api"
	"object-storage-golang/storage"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "modernc.org/sqlite"
)

// @title           Object Storage S3 API
// @version         0.1
// @description     A small S3-compatible object storage API.
// @host            localhost:8080
// @BasePath        /
// @schemes         http
func main() {
	ctx := context.Background()

	db, err := sql.Open("sqlite", "./data/metadata.db")
	if err != nil {
		panic(fmt.Errorf("open sqlite: %w", err))
	}
	defer db.Close()

	metadataStore := object.NewSQLiteMetadataStore(db)
	if err := metadataStore.Initialize(ctx); err != nil {
		panic(fmt.Errorf("initialize metadata store: %w", err))
	}
	// Create a new request multiplexer (router)
	blobStore := &storage.FileSystemBlobStore{
		Root: "./data",
	}
	service := object.NewObjectService(blobStore, metadataStore)
	mux := http.NewServeMux()

	// Register the handler function for the root path
	mux.HandleFunc("GET /{bucket}/{key...}", s3api.S3GetObjectHandler(service))
	mux.HandleFunc("PUT /{bucket}/{key...}", s3api.S3PutObjectHandler(service))
	mux.HandleFunc("DELETE /{bucket}/{key...}", s3api.S3DeleteObjectHandler(service))
	mux.Handle("GET /openapi/", http.StripPrefix("/openapi/", http.FileServer(http.Dir("./openapi"))))
	mux.Handle(
		"GET /swagger/",
		httpSwagger.Handler(
			httpSwagger.URL("/openapi/swagger.json"),
		),
	)

	fmt.Println("Server starting on http://localhost:8080")

	// Start the server and listen on port 8080
	if err := http.ListenAndServe(":8080", httpmiddleware.RejectUnsafePaths(s3api.WriteS3HTTPError)(mux)); err != nil {
		panic(err)
	}
}
