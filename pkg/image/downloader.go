// Package image is an image uploader
package image

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/tritonmedia/identifier/pkg/providerapi"
)

// Downloader is an image downloader that is content aware
type Downloader struct{}

// NewDownloader creates a new image downloader
func NewDownloader() *Downloader {
	return &Downloader{}
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

	data := make([]byte, 0)

	switch contentType {
	case "image/png":
		data = b
	case "image/jpeg":
		img, err := jpeg.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, errors.Wrap(err, "unable to decode jpeg")
		}

		buf := new(bytes.Buffer)
		if err := png.Encode(buf, img); err != nil {
			return nil, errors.Wrap(err, "unable to encode png")
		}

		data = buf.Bytes()
	default:
		return nil, fmt.Errorf("unsupported image protocol %v", contentType)
	}

	img, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dimensions")
	}

	i.Resolution = fmt.Sprintf("%dx%d", img.Width, img.Height)

	return &data, nil
}
