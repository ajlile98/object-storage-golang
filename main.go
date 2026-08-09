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
	"os"

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
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	db, err := openDB(ctx, "./data/metadatastore")
	if err != nil {
		return err
	}
	defer db.Close()

	metadataStore := object.NewSQLiteMetadataStore(db)
	if err := metadataStore.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize metadata store: %w", err)
	}

	blobStore := &storage.FileSystemBlobStore{
		Root: "./data/blobstore",
	}

	service := object.NewObjectService(blobStore, metadataStore)

	mux := setupRoutes(service)

	fmt.Println("Server starting on http://localhost:8080")

	// Start the server and listen on port 8080
	if err := http.ListenAndServe(":8080", httpmiddleware.RejectUnsafePaths(s3api.WriteS3HTTPError)(mux)); err != nil {
		return err
	}
	return nil
}

func openDB(ctx context.Context, path string) (*sql.DB, error) {
	os.MkdirAll("./data", 0o755)
	db, err := sql.Open("sqlite", "./data/metadata.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)")
	if err != nil {
		return &sql.DB{}, fmt.Errorf("open sqlite: %w", err)
	}
	return db, nil
}

func setupRoutes(service *object.ObjectService) http.Handler {
	mux := http.NewServeMux()
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
	return mux
}
