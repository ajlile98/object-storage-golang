package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

type FileSystemBlobStore struct {
	Root string
}

var ErrBlobIdConflict = errors.New("blob id conflicts with an existing blob id")
var ErrInvalidBlobId = errors.New("invalid blob id")

func (store *FileSystemBlobStore) Put(
	ctx context.Context,
	blobID string,
	data io.Reader,
) (size int64, checksum string, err error) {
	// Stream data to a file under store.root
	if err := validateID(blobID); err != nil {
		return 0, "", err
	}
	objectpath := store.blobPath(blobID)

	size, checksum, tmpFilePath, err := store.stageBlob(ctx, objectpath, data)
	if err != nil {
		return 0, "", err
	}

	if err := store.commitBlob(tmpFilePath, objectpath); err != nil {
		return 0, "", err
	}

	return size, checksum, nil
}

func (store *FileSystemBlobStore) stageBlob(
	ctx context.Context,
	objectPath string,
	data io.Reader,
) (size int64, checksum, tmpFilePath string, err error) {
	tmpdir := filepath.Join(store.Root, "tmp")

	if err := ctx.Err(); err != nil {
		return 0, "", "", err
	}

	if err := os.MkdirAll(tmpdir, 0o755); err != nil {
		return 0, "", "", fmt.Errorf("create tmp directory: %w", err)
	}

	tmpfile, err := os.CreateTemp(tmpdir, "blob-*")
	if err != nil {
		return 0, "", "", fmt.Errorf("create tmp object file: %w", err)
	}
	hasher := sha256.New()
	writer := io.MultiWriter(tmpfile, hasher)

	size, err = io.Copy(writer, data)
	if err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return 0, "", "", fmt.Errorf("write object data: %w", err)
	}
	if err := tmpfile.Sync(); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return 0, "", "", err
	}
	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return 0, "", "", err
	}

	checksum = fmt.Sprintf("%x", hasher.Sum(nil))
	return size, checksum, tmpfile.Name(), nil
}

func (store *FileSystemBlobStore) commitBlob(tmpFilePath, objectPath string) error {
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		return fmt.Errorf("create object directories: %w", err)
	}

	if err := os.Rename(tmpFilePath, objectPath); err != nil {
		os.Remove(tmpFilePath)
		return err
	}

	// fsync the parent dir so the rename itself survives a crash.
	dir, err := os.Open(filepath.Dir(objectPath))
	if err != nil {
		return fmt.Errorf("open parent dir for sync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		dir.Close()
		return fmt.Errorf("sync parent dir: %w", err)
	}

	return nil
}

func (store *FileSystemBlobStore) Get(
	ctx context.Context,
	blobID string,
) (io.ReadCloser, error) {
	// open and return the local file
	if err := validateID(blobID); err != nil {
		return nil, err
	}
	objectpath := store.blobPath(blobID)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.Open(objectpath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	return file, nil
}

func (store *FileSystemBlobStore) Delete(
	ctx context.Context,
	blobID string,
) error {
	// delete local file
	if err := validateID(blobID); err != nil {
		return err
	}
	objectpath := store.blobPath(blobID)

	if err := ctx.Err(); err != nil {
		return err
	}

	err := os.Remove(objectpath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	return nil
}

func (store *FileSystemBlobStore) blobPath(blobID string) string {
	return filepath.Join(store.Root, blobID[0:2], blobID[2:4], blobID)
}

var blobIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func validateID(blobID string) error {
	if !blobIDPattern.MatchString(blobID) {
		return ErrInvalidBlobId
	}
	return nil
}
