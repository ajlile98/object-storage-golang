package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"object-storage-golang/httpmiddleware"
	"object-storage-golang/object"
	"object-storage-golang/s3api"
	"object-storage-golang/storage"
	"os"
	"path/filepath"

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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	db, err := openDB(ctx, "./data/metadatastore/metadatastore.db")
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

	handler := httpmiddleware.LogRequests(
		httpmiddleware.RejectUnsafePaths(s3api.WriteS3HTTPError)(mux),
	)

	slog.Info("server starting", "addr", "http://localhost:8080")

	return http.ListenAndServe(":8080", handler)
}

func openDB(ctx context.Context, path string) (*sql.DB, error) {
	perms := os.FileMode(0o755)
	err := os.MkdirAll(filepath.Dir(path), perms)
	if err != nil {
		return nil, fmt.Errorf("makedirall path=%s perms=%s: %w", path, perms, err)
	}
	db, err := sql.Open("sqlite", fmt.Sprintf(
		"%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)",
		path,
	),
	)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
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
