package image

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tritonmedia/identifier/pkg/providerapi"
)

func TestImageNewDownloader(t *testing.T) {
	d := NewDownloader()
	assert.NotNil(t, d)
}

func TestImageDownloaderDownload(t *testing.T) {
	d := NewDownloader()
	b, err := d.DownloadImage(&providerapi.Image{
		// By POV-Ray source code - Own work: Rendered in POV-Ray by user:ed_g2s., CC BY-SA 3.0, https://commons.wikimedia.org/w/index.php?curid=221157
		URL: "https://upload.wikimedia.org/wikipedia/commons/4/47/PNG_transparency_demonstration_1.png",
	})

	assert.NoError(t, err)
	assert.NotNil(t, b, "returned nil image")
}

func TestImageDownloaderDownloadConvertPng(t *testing.T) {
	d := NewDownloader()
	b, err := d.DownloadImage(&providerapi.Image{
		// By Felis_silvestris_silvestris.jpg: Michael Gäblerderivative work: AzaToth - Felis_silvestris_silvestris.jpg, CC BY 3.0, https://commons.wikimedia.org/w/index.php?curid=16857750
		URL: "https://upload.wikimedia.org/wikipedia/commons/e/e9/Felis_silvestris_silvestris_small_gradual_decrease_of_quality.png",
	})

	ct := http.DetectContentType(*b)

	assert.NoError(t, err)
	assert.NotNil(t, b, "returned nil image")
	assert.Equal(t, "image/png", ct, "got back wrong type when converting to png")
}
