package events

import (
	"github.com/tritonmedia/identifier/pkg/image"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	"github.com/tritonmedia/identifier/pkg/storageapi"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

// ProcessorConfig is an event processor config
type ProcessorConfig struct {
	Providers       map[api.Media_MetadataType]providerapi.Fetcher
	DB              storageapi.Provider
	ImageDownloader *image.Downloader
	ImageUploader   *image.Uploader
}
