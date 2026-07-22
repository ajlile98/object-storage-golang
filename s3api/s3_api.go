package s3api

import (
	"encoding/xml"
	"errors"
	"net/http"
	"object-storage-golang/object"
	"object-storage-golang/storage"
)

type s3ErrorResponse struct {
	XMLName  xml.Name `xml:"Error"`
	Code     string   `xml:"Code"`
	Message  string   `xml:"Message"`
	Resource string   `xml:"Resource,omitempty"`
}

type ErrorResponder func(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	code string,
	message string,
)

func S3PutObjectHandler(service *object.ObjectService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucket := r.PathValue("bucket")
		key := r.PathValue("key")
		resource := r.URL.Path
		if bucket == "" || key == "" {
			writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "bucket and object key are required", resource)
			return
		}

		_, checksum, err := service.Blobs.Put(r.Context(), bucket+"/"+key, r.Body)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrInvalidKey):
				writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "invalid object key", resource)
			case errors.Is(err, storage.ErrKeyConflict):
				writeS3Error(w, http.StatusConflict, "InvalidRequest", "object key conflicts with an existing path", resource)
			default:
				writeS3Error(w, http.StatusInternalServerError, "InternalError", "failed to store object", resource)
			}
			return
		}

		w.Header().Set("ETag", `"`+checksum+`"`)
		w.WriteHeader(http.StatusOK)
	}
}

func writeS3Error(w http.ResponseWriter, status int, code, message, resource string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_ = xml.NewEncoder(w).Encode(s3ErrorResponse{
		Code:     code,
		Message:  message,
		Resource: resource,
	})
}
