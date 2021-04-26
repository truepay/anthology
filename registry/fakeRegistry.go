package registry

import (
	"gocloud.dev/blob/memblob"
)

func NewFakeRegistry() Registry {
	bucket := memblob.OpenBucket(nil)
	return newBlobRegistry(bucket)
}
