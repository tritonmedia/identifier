// Package image is an image uploader
package image

import (
	"fmt"
	"hash/crc64"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	"gopkg.in/h2non/bimg.v1"
)

// Downloader is an image downloader that is content aware
type Downloader struct {
	crctable *crc64.Table
}

// NewDownloader creates a new image downloader
func NewDownloader() *Downloader {
	return &Downloader{
		crctable: crc64.MakeTable(0xC96C5795D7870F42),
	}
}

// DownloadImage downloads an image in memory, converting it to png
// also reads the resolution and sets it on the image struct
func (d *Downloader) DownloadImage(i *providerapi.Image) (*[]byte, error) {
	res, err := http.Get(i.URL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	contentType := http.DetectContentType(b)

	var data []byte

	switch contentType {
	case "image/png":
		data = b
	case "image/jpeg":
		img, err := bimg.NewImage(b).Convert(bimg.PNG)
		if err != nil {
			return nil, errors.Wrap(err, "unable to convert jpeg to png")
		}

		data = img
	default:
		return nil, fmt.Errorf("unsupported image protocol %v", contentType)
	}

	size, err := bimg.NewImage(data).Size()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dimensions")
	}

	i.Resolution = fmt.Sprintf("%dx%d", size.Width, size.Height)

	ci := crc64.Checksum(data, d.crctable)
	i.Checksum = strconv.FormatUint(ci, 16)

	return &data, nil
}
