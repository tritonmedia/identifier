package storageapi

import "github.com/tritonmedia/identifier/pkg/providerapi"

// Provider is an interface for storage providers to implement
type Provider interface {

	// NewSeries create a new series entry in the database
	NewSeries(*providerapi.Series) error

	// NewEpisodes adds a list of episodes to the data base
	NewEpisodes(*providerapi.Series, []providerapi.Episode) error

	// NewImage creates a new image
	NewImage(*providerapi.Series, *providerapi.Image) (string, error)

	// NewEpisodeFile creates a new episode file
	NewEpisodeFile(e *providerapi.Episode, key, quality string) (string, error)

	// FindEpisodeID finds an episode's ID by episode and season number
	FindEpisodeID(mediaID string, episode, season int) (string, error)
}
