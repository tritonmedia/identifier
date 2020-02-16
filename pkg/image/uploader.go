// Package image is an image uploader
package image

import (
	"bytes"
	"fmt"

	"github.com/minio/minio-go/v6"
	"github.com/tritonmedia/identifier/pkg/providerapi"
)

// Uploader is an image uploader to S3
type Uploader struct {
	s3client *minio.Client
	bucket   string
}

// NewUploader creates a new image uploader
func NewUploader(m *minio.Client, bucketName string) *Uploader {
	return &Uploader{
		s3client: m,
		bucket:   bucketName,
	}
}

// UploadImage uploads an image to s3
func (u *Uploader) UploadImage(mediaID, imageID string, image *[]byte, i *providerapi.Image) error {
	key := fmt.Sprintf("images/%s/%s.png", mediaID, imageID)

	if i.Resolution == "" {
		return fmt.Errorf("image has no resolution")
	}

	img := *image
	if _, err := u.s3client.PutObject(u.bucket, key, bytes.NewReader(img), int64(len(img)), minio.PutObjectOptions{}); err != nil {
		return err
	}

	// set the s3 key
	i.Key = key
	return nil
}
