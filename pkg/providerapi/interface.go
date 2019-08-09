package providerapi

import "time"

// ImageType is the type of image an image is
type ImageType int

const (
	// ImagePoster is a poster type image
	ImagePoster ImageType = 0

	// ImageBanner is a banner type image
	ImageBanner ImageType = 1

	// ImageBackground is a background type image
	ImageBackground ImageType = 2
)

// Image is a image provided by a metadata provider
type Image struct {
	// Type of this media
	Type ImageType

	// URL is the URL to obtain this media
	URL string

	// Rating of this image (1-10)
	Rating int

	// Resolution of this image
	Resolution string

	// Thumbnail URL (if applicable)
	ThumbnailURL string
}

// Metadata is an struct that should be returned by a provider for media
type Metadata struct {
	// Title of this media
	Title string

	// Provider that returned this, should match the v1.media metadata entry
	Provider int

	// ProviderID is the id that could be used to cross reference this later
	ProviderID string

	// Overview of the media
	Overview string

	// 1-10 rating of a show
	Rating int

	// If applicable, the network of this media
	Network string

	// FirstAired is when this first aired
	FirstAired *time.Time

	// FinishedAiring is when this finished airing, if applicable
	FinishedAriring *time.Time

	// Genre types
	Genre []string

	// Time of day this show airs, should be HH:MM (24hr)
	Airs string

	// Day of the week this show airs
	AirDayOfWeek string

	// Runtime, average runtime of this media if applicable
	Runtime int

	// Images are images provided for this media
	Images []Image
}

// ProviderFetcher is an interface that a provider should implement in order to be able to provide
// metadata
type ProviderFetcher interface {
	Get(id string) *Metadata
}
