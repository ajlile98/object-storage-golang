package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type SQLiteStore struct {
	db *sql.DB
}

var ErrObjectNotFound = errors.New("object not found")

func (s *SQLiteStore) CreateObject(
	ctx context.Context,
	object ObjectMetadata,
) (ObjectMetadata, error) {
	row := s.db.QueryRowContext(
		ctx,
		`INSERT INTO objects (
            bucket_name, object_key, version_id, blob_id,
            size, content_type, checksum, state
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING
		bucket_name,
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

func (s *SQLiteStore) CompleteObjectUpload(
	ctx context.Context,
	object ObjectMetadata,
) (ObjectMetadata, error) {
	row := s.db.QueryRowContext(
		ctx,
		`UPDATE objects
		SET size = ?,
			checksum = ?,
			state = ?
		WHERE bucket_name = ? 
			AND object_key = ? 
			AND version_id = ?
			AND state = ?
		RETURNING
            bucket_name,
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

func (s *SQLiteStore) GetObject(
	ctx context.Context,
	bucket, key string,
) (ObjectMetadata, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT bucket_name, object_key, version_id, blob_id,
            size, content_type, checksum, created_at, state
		FROM objects
		WHERE bucket_name = ? 
			AND object_key = ?
		ORDER BY created_at DESC
		LIMIT 1`,
		bucket,
		key,
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
	if metadata.State != ObjectStateReady {
		return ObjectMetadata{}, ErrObjectNotFound
	}
	return metadata, nil
}

func (s *SQLiteStore) MarkObjectDeleted(
	ctx context.Context,
	bucket, key string,
) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE objects SET state = ?
		WHERE rowid = (
			SELECT rowid FROM objects
			WHERE bucket_name = ? AND object_key = ? AND state = ?
			ORDER BY created_at DESC
			LIMIT 1
		)`,
		ObjectStateDeleted,
		bucket,
		key,
		ObjectStateReady,
	)

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

func (s *SQLiteStore) Initialize(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS buckets (
			bucket_name TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
			state TEXT NOT NULL DEFAULT 'ready',
			PRIMARY KEY (name)
		);
  
		CREATE TABLE IF NOT EXISTS objects (
            bucket_name TEXT NOT NULL,
            object_key TEXT NOT NULL,
            version_id TEXT NOT NULL,
            blob_id TEXT NOT NULL,
            size INTEGER NOT NULL,
            content_type TEXT NOT NULL,
            checksum TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
            state TEXT NOT NULL DEFAULT 'pending',
            PRIMARY KEY (bucket_name, object_key, version_id)
        );
    `)
	if err != nil {
		return fmt.Errorf("create objects table: %w", err)
	}
	return nil
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) CreateBucket(ctx context.Context, name string) {

}
