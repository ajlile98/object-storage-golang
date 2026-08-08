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

var ErrObjectNotFound = errors.New("object not found")

func (store *SQLiteMetadataStore) Create(
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
		return ObjectMetadata{}, fmt.Errorf("create pending object: %w", err)
	}

	return persisted, nil
}

func (store *SQLiteMetadataStore) CompleteUpload(
	ctx context.Context,
	object ObjectMetadata,
) (ObjectMetadata, error) {
	row := store.db.QueryRowContext(
		ctx,
		`UPDATE objects
		SET size = ?,
			checksum = ?,
			state = ?
		WHERE bucket = ? 
			AND object_key = ? 
			AND version_id = ?
			AND state = ?
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
		object.Size,
		object.Checksum,
		ObjectStateReady,
		object.Bucket,
		object.Key,
		object.VersionID,
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
		if errors.Is(err, sql.ErrNoRows) {
			return ObjectMetadata{}, ErrObjectNotFound
		}
		return ObjectMetadata{}, fmt.Errorf("complete upload: %w", err)
	}
	return persisted, nil
}

func (store *SQLiteMetadataStore) Get(
	ctx context.Context,
	bucket, key string,
) (ObjectMetadata, error) {
	row := store.db.QueryRowContext(
		ctx,
		`SELECT bucket, object_key, version_id, blob_id,
            size, content_type, checksum, created_at, state
		FROM objects
		WHERE bucket = ? 
			AND object_key = ?
			AND state = ?
		ORDER BY created_at DESC
		LIMIT 1`,
		bucket,
		key,
		ObjectStateReady,
	)
	var metadata ObjectMetadata
	err := row.Scan(
		&metadata.Bucket,
		&metadata.Key,
		&metadata.VersionID,
		&metadata.BlobID,
		&metadata.Size,
		&metadata.ContentType,
		&metadata.Checksum,
		&metadata.CreatedAt,
		&metadata.State,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ObjectMetadata{}, ErrObjectNotFound
	}
	if err != nil {
		return ObjectMetadata{}, fmt.Errorf("get object: %w", err)
	}
	return metadata, nil
}

// func (store *SQLiteMetadataStore) GetVersion(
// 	ctx context.Context,
// 	bucket, key string,
// ) (ObjectMetadata, error) {
// 	return ObjectMetadata{}, nil
// }

func (store *SQLiteMetadataStore) MarkDeleted(
	ctx context.Context,
	bucket, key, version_id string,
) error {
	result, err := store.db.ExecContext(ctx, `
		UPDATE objects 
		SET state = ?
		WHERE bucket = ? 
			AND object_key = ? 
			AND version_id = ?
			AND state = ?`,
		ObjectStateDeleted,
		bucket,
		key,
		version_id,
		ObjectStateReady,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrObjectNotFound
	}
	if err != nil {
		return fmt.Errorf("mark object deleted: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted objects: %w", err)
	}
	if affected == 0 {
		return ErrObjectNotFound
	}

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
			created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
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
