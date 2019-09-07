package storageapi

import (
	"github.com/oz/osdb"
	"github.com/tritonmedia/identifier/pkg/providerapi"
)

// Provider is an interface for storage providers to implement
type Provider interface {

	// NewSeries create a new series entry in the database
	NewSeries(*providerapi.Series) error

	// NewEpisodes adds a list of episodes to the data base
	NewEpisodes(*providerapi.Series, []providerapi.Episode) error

	// NewImage creates a new image
	NewImage(*providerapi.Series, *providerapi.Image) (string, error)

	// NewEpisodeImage creates a new image
	NewEpisodeImage(*providerapi.Episode, *providerapi.Image) (string, error)

	// NewEpisodeFile creates a new episode file
	NewEpisodeFile(e *providerapi.Episode, key, quality string) (string, error)

	// FindEpisodeID finds an episode's ID by episode and season number
	FindEpisodeID(mediaID string, episode, season int) (string, error)

	// GetSeriesByID returns a series by ID
	GetSeriesByID(mediaID string) (providerapi.Series, error)

	// GetEpisodeByID returns an episode by ID
	GetEpisodeByID(s *providerapi.Series, episodeID string) (providerapi.Episode, error)

	// NewSubtitle creates a new subtitle
	NewSubtitle(s *providerapi.Series, e *providerapi.Episode, sub *osdb.Subtitle) (string, string, error)
}
