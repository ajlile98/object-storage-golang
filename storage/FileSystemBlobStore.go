package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

type FileSystemBlobStore struct {
	Root string
}

var ErrKeyConflict = errors.New("object key conflicts with an existing object path")
var ErrInvalidKey = errors.New("invalid object key")

func (store *FileSystemBlobStore) Put(
	ctx context.Context,
	key string,
	data io.Reader,
) (size int64, checksum string, err error) {
	// Stream data to a file under store.root
	if err := validateKey(key); err != nil {
		return 0, "", err
	}
	objectpath := filepath.Join(store.Root, filepath.FromSlash(key))

	if err := ctx.Err(); err != nil {
		return 0, "", err
	}

	if err := os.MkdirAll(filepath.Dir(objectpath), 0o755); err != nil {
		return 0, "", fmt.Errorf("create object directories: %w", err)
	}

	file, err := os.Create(objectpath)
	if err != nil {
		if errors.Is(err, syscall.ENOTDIR) || errors.Is(err, syscall.EISDIR) {
			return 0, "", fmt.Errorf("%w: %s", ErrKeyConflict, key)
		}
		return 0, "", fmt.Errorf("create object file: %w", err)
	}
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	defer file.Close()

	size, err = io.Copy(writer, data)
	if err != nil {
		return 0, "", fmt.Errorf("write object data: %w", err)
	}

	checksum = fmt.Sprintf("%x", hasher.Sum(nil))
	return size, checksum, nil
}

func (store *FileSystemBlobStore) Get(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	// open and return the local file
	if err := validateKey(key); err != nil {
		return nil, err
	}
	objectpath := filepath.Join(store.Root, filepath.FromSlash(key))

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
	key string,
) error {
	// delete local file
	if err := validateKey(key); err != nil {
		return err
	}
	objectpath := filepath.Join(store.Root, filepath.FromSlash(key))

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

func validateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	// Object keys always use '/', even when the server runs on Windows.
	if strings.Contains(key, `\`) {
		return ErrInvalidKey
	}

	// Reject absolute paths such as "/secret.txt".
	if strings.HasPrefix(key, "/") {
		return ErrInvalidKey
	}

	// Reject Windows paths such as "C:/secret.txt".
	if filepath.IsAbs(key) || filepath.VolumeName(key) != "" {
		return ErrInvalidKey
	}

	for _, segment := range strings.Split(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return ErrInvalidKey
		}
	}

	// Defense in depth: cleaning must not change the accepted key.
	if path.Clean(key) != key {
		return ErrInvalidKey
	}

	return nil
}
