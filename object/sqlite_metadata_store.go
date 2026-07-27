package object

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type SQLiteMetadataStore struct {
	db *sql.DB
}

var ErrObjectNotFound = errors.New("Object Not Found")
var ErrObjectNotCreated = errors.New("Object Not Created")

func (store *SQLiteMetadataStore) CreatePending(
	ctx context.Context,
	object ObjectMetadata,
) (ObjectMetadata, error) {
	row := store.db.QueryRowContext(
		ctx,
		`INSERT INTO objects (
            bucket, object_key, version_id, blob_id,
            size, content_type, checksum, state
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING
        	bucket,
        	object_key,
        	version_id,
        	blob_id,
        	size,
        	content_type,
        	checksum,
        	created_at,
        	state`,
		object.Bucket,
		object.Key,
		object.VersionID,
		object.BlobID,
		object.Size,
		object.ContentType,
		object.Checksum,
		ObjectStatePending,
	)

	var persisted ObjectMetadata
	if err := row.Scan(
		&persisted.Bucket,
		&persisted.Key,
		&persisted.VersionID,
		&persisted.BlobID,
		&persisted.Size,
		&persisted.ContentType,
		&persisted.Checksum,
		&persisted.CreatedAt,
		&persisted.State,
	); err != nil {
		return ObjectMetadata{}, fmt.Errorf("create pending object %w", err)
	}

	return persisted, nil
}

func (store *SQLiteMetadataStore) MarkReady(
	ctx context.Context,
	bucket, key, version string,
) error {
	result, err := store.db.ExecContext(
		ctx,
		`UPDATE objects
		SET state = ?
		WHERE bucket = ? AND object_key = ? AND version_id = ?`,
		ObjectStateReady,
		bucket,
		key,
		version,
	)
	if err != nil {
		return fmt.Errorf("mark object read: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count updated objects %w", err)
	}
	if affected == 0 {
		return ErrObjectNotFound
	}

	return nil
}

func (store *SQLiteMetadataStore) Get(
	ctx context.Context,
	bucket, key string,
) (ObjectMetadata, error) {
	return ObjectMetadata{}, nil
}

func (store *SQLiteMetadataStore) GetVersion(
	ctx context.Context,
	bucket, key string,
) (ObjectMetadata, error) {
	return ObjectMetadata{}, nil
}

func (store *SQLiteMetadataStore) MarkDeleted(
	ctx context.Context,
	bucket, key string,
) error {
	return nil
}

func (store *SQLiteMetadataStore) Initialize(ctx context.Context) error {
	_, err := store.db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS objects (
            bucket TEXT NOT NULL,
            object_key TEXT NOT NULL,
            version_id TEXT NOT NULL,
            blob_id TEXT NOT NULL,
            size INTEGER NOT NULL,
            content_type TEXT NOT NULL,
            checksum TEXT NOT NULL,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            state TEXT NOT NULL DEFAULT 'pending',
            PRIMARY KEY (bucket, object_key, version_id)
        )
    `)
	if err != nil {
		return fmt.Errorf("create objects table: %w", err)
	}
	return nil
}

func NewSQLiteMetadataStore(db *sql.DB) *SQLiteMetadataStore {
	return &SQLiteMetadataStore{db: db}
}
