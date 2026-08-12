package s3api

import (
	"encoding/xml"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"object-storage-golang/object"
	"object-storage-golang/storage"
	"strconv"
)

type s3ErrorResponse struct {
	XMLName  xml.Name `xml:"Error"`
	Code     string   `xml:"Code"`
	Message  string   `xml:"Message"`
	Resource string   `xml:"Resource,omitempty"`
}

// formatETag wraps a checksum in the quoted-string form required by the HTTP ETag header.
func formatETag(checksum string) string {
	return `"` + checksum + `"`
}

func S3PutObjectHandler(service *object.ObjectService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucket := r.PathValue("bucket")
		key := r.PathValue("key")
		resource := r.URL.Path
		if bucket == "" || key == "" {
			writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "bucket and object key are required", resource)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// _, checksum, err := service.Blobs.Put(r.Context(), bucket+"/"+key, r.Body)
		metadata, err := service.Upload(
			r.Context(),
			bucket,
			key,
			contentType,
			r.Body,
		)
		if err != nil {
			slog.Error(
				"put object failed",
				"error", err,
				"bucket", bucket,
				"key", key,
			)
			switch {
			case errors.Is(err, storage.ErrInvalidBlobId):
				writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "invalid object key", resource)
			case errors.Is(err, storage.ErrBlobIdConflict):
				writeS3Error(w, http.StatusConflict, "InvalidRequest", "object key conflicts with an existing path", resource)
			default:
				writeS3Error(w, http.StatusInternalServerError, "InternalError", "failed to store object", resource)
			}
			return
		}

		w.Header().Set("ETag", formatETag(metadata.Checksum))
		w.WriteHeader(http.StatusOK)
	}
}

func S3GetObjectHandler(service *object.ObjectService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucket := r.PathValue("bucket")
		key := r.PathValue("key")
		resource := r.URL.Path
		if bucket == "" || key == "" {
			writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "bucket and object key are required", resource)
			return
		}

		obj, err := service.Read(r.Context(), bucket, key)
		if err != nil {
			// Translate the error to an S3 XML response.
			if errors.Is(err, object.ErrObjectNotFound) {
				writeS3Error(
					w,
					http.StatusNotFound,
					"NoSuchKey",
					"the specified key does not exist",
					resource,
				)
				return
			}
			writeS3Error(
				w,
				http.StatusInternalServerError,
				"InternalError",
				"failed to read object",
				resource,
			)
			return
		}

		defer obj.Body.Close()

		contentType := obj.Metadata.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)

		w.Header().Set("Content-Length", strconv.FormatInt(obj.Metadata.Size, 10))

		w.Header().Set("ETag", formatETag(obj.Metadata.Checksum))

		if _, err := io.Copy(w, obj.Body); err != nil {
			// The client may have disconnected during the stream.
			// At this point headers/body may already be sent, so do not write an S3 error response.
			return
		}

	}
}

func S3DeleteObjectHandler(service *object.ObjectService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucket := r.PathValue("bucket")
		key := r.PathValue("key")
		resource := r.URL.Path
		if bucket == "" || key == "" {
			writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "bucket and object key are required", resource)
			return
		}

		err := service.Delete(r.Context(), bucket, key)
		if err != nil {
			writeS3Error(
				w,
				http.StatusInternalServerError,
				"InternalError",
				"failed to delete object",
				resource,
			)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func writeS3Error(
	w http.ResponseWriter,
	status int,
	code,
	message,
	resource string,
) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_ = xml.NewEncoder(w).Encode(s3ErrorResponse{
		Code:     code,
		Message:  message,
		Resource: resource,
	})
}

func WriteS3HTTPError(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	code string,
	message string,
) {
	writeS3Error(w, status, code, message, r.URL.Path)
}
