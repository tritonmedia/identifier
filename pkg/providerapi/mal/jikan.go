// Package mal is a providerapi.Fetcher implementation for MAL using jikan
package mal

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	jikan "github.com/tritonmedia/jikan-go"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

const (
	// Provider is the provider enum entry for this provider
	Provider = int(api.Media_MAL)
)

// Client is a IMDB TVDB-wrapped client
type Client struct{}

// NewClient returns a new imdb client
func NewClient() *Client {
	return &Client{}
}

// GetSeries returns a series by ID
func (c *Client) GetSeries(mediaID string, mediaType api.Media_MediaType, strid string) (providerapi.Series, error) {
	id, err := strconv.Atoi(strid)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to convert id to an int")
	}

	a, err := jikan.GetAnimeInfo(id)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to get mal (jikan) anime")
	}

	// TODO(jaredallard): find non finished status
	var status providerapi.SeriesStatus
	switch a.Status {
	case "Finished Airing":
		status = providerapi.SeriesEnded
		break
	}

	genres := make([]string, len(a.Genres))
	for i, genere := range a.Genres {
		genres[i] = genere.Name
	}

	images := make([]providerapi.Image, 2)
	images[0] = providerapi.Image{
		Type:   providerapi.ImagePoster,
		URL:    a.ImageURL,
		Rating: 10,
	}

	s := providerapi.Series{
		ID:         mediaID,
		Type:       mediaType,
		Title:      a.TitleEnglish,
		Provider:   Provider,
		ProviderID: strid,
		Overview:   a.Synopsis,
		Rating:     float32(a.Score),
		FirstAired: a.Aired.From,
		Status:     status,
		Network:    a.Broadcast,
		Genre:      genres,

		// These aren't exposed by Jikan
		Airs:         "",
		AirDayOfWeek: "",
		Runtime:      0,

		Images: images,
	}
	return s, nil
}

// GetEpisodes returns episodes in a series
// MyAnimeList joins seasons together as "Sequels" so we have to fetch episodes
// from all of those medias
func (c *Client) GetEpisodes(s *providerapi.Series) ([]providerapi.Episode, error) {
	// there are no episodes in a tv show, but we create a single -1 number episode
	if s.Type == api.Media_MOVIE {
		return []providerapi.Episode{
			providerapi.Episode{
				Number: -1,
			},
		}, nil
	}

	id, err := strconv.Atoi(s.ProviderID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert id to an int")
	}

	a, err := jikan.GetAnimeInfo(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get mal (jikan) anime")
	}

	_ = a

	eps, err := jikan.GetAnimeEpisodes(id, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get eps")
	}

	newEps := make([]providerapi.Episode, len(eps.Episodes))
	for i, e := range eps.Episodes {

		newEps[i] = providerapi.Episode{
			Number:       int64(e.EpisodeID),
			SeasonNumber: int64(e.EpisodeID),
			Season:       0,
			Name:         e.Title,
			Overview:     "Not Currently Provided by Jikan",
			Aired:        e.Aired,
			Rating:       10,
		}
	}

	return newEps, nil
}
