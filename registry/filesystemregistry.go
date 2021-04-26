package registry

import (
	"os"
	"strings"

	"github.com/erikvanbrakel/anthology/app"
	"github.com/sirupsen/logrus"
	"gocloud.dev/blob/fileblob"
)

func NewFilesystemRegistry(options app.FileSystemOptions) (Registry, error) {
	basePath := options.BasePath
	if !strings.HasSuffix(basePath, string(os.PathSeparator)) {
		basePath = basePath + string(os.PathSeparator)
	}

	logrus.Infof("Using Filesystem Registry with basepath %s", basePath)

	bucket, err := fileblob.OpenBucket(basePath, nil)
	if err != nil {
		return nil, err
	}
	return newBlobRegistry(bucket), nil
}
