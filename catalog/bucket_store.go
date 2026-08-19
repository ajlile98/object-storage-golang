package catalog

import "time"

type Bucket struct {
	Name      string
	CreatedAt time.Time
	State     BucketState
}

type BucketState string

const (
	BucketStateReady    BucketState = "ready"
	BucketStateDeleting BucketState = "deleting"
)
