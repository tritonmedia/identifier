package providerapi

import (
	"time"

	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

// ImageType is the type of image an image is
type ImageType string

// SeriesStatus is the status of a series
type SeriesStatus string

const (
	// ImagePoster is a poster type image
	ImagePoster ImageType = "poster"

	// ImageBackground is a background type image
	ImageBackground ImageType = "background"

	// ImageActor is an actor image
	ImageActor ImageType = "actor"

	// ImageThumbnail is a thumbnail for an episode
	ImageThumbnail ImageType = "thumbnail"

	// SeriesAiring denotes a series is still airing
	SeriesAiring SeriesStatus = "Airing"

	// SeriesEnded denotes a series is finished
	SeriesEnded SeriesStatus = "Ended"
)

// Image is a image provided by a metadata provider
type Image struct {
	// Type of this media
	Type ImageType

	// Key is the S3 key of this image, not expected to be set
	// by a provider
	Key string

	// Checksum is a CRC64 checksum provided by the image package after download
	Checksum string

	// URL is the URL to obtain this media
	URL string

	// Rating of this image (1-10)
	Rating float64

	// Resolution of this image
	Resolution string

	// Thumbnail URL (if applicable)
	ThumbnailURL string
}

// Actor is a actor provided by a metadata provider
type Actor struct {
	// Name of the actor
	Name string

	// Role this actor played in this series
	Role string

	// Images is a list of images for this actor
	Images []Image
}

// Series is an struct that should be returned by a provider for a media series
type Series struct {
	// Title of this media
	Title string

	// ID of this media, set by identifier
	ID string

	// Type of this media, set by providerapi
	Type api.Media_MediaType

	// Provider that returned this, should match the v1.media metadata entry
	Provider int

	// ProviderID is the id that could be used to cross reference this later
	ProviderID string

	// Overview of the media
	Overview string

	// 1-10 rating of a show
	Rating float32

	// If applicable, the network of this media
	// TODO(jaredallard): some providers store network ids, use these?
	Network string

	// FirstAired is when this first aired
	FirstAired time.Time

	// Status
	Status SeriesStatus

	// Genre types
	// TODO(jaredallard): case these to enumerable types
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

// Episode is an episode of a series
// TODO(jaredallard): provider actors on this
type Episode struct {
	// ID is the ID of this episode, set by identifier
	ID string

	// Number is the absolute number of a episode in a series, is not
	// bound by the current season
	Number int64

	// Season Number is the number of this episode in a season
	SeasonNumber int64

	// Season this episode is apart of
	Season int

	// Name of this episode
	Name string

	// Overview of this episode
	Overview string

	// Aired is when a this episode aired
	Aired time.Time

	// Rating (1-10) of this episode
	Rating float32

	// Thumbnail should be a ImageThumb image
	Thumb Image
}

// Fetcher is an interface that a provider should implement in order to be able to provide
// metadata
type Fetcher interface {
	// GetSeries returns a series by provider ID
	GetSeries(mediaID string, mediaType api.Media_MediaType, id string) (Series, error)

	// GetEpisodes returns all episodes in a series, if it's a movie it should return
	// a single episode.
	GetEpisodes(*Series) ([]Episode, error)
}
