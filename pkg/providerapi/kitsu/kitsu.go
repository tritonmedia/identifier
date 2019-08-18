// Package kitsu is a providerapi.Fetcher implementation for kitsu
package kitsu

import (
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	"github.com/tritonmedia/kitsu"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

const (
	// Provider is the provider enum entry for this provider
	Provider = int(api.Media_KITSU)
)

// Client is a IMDB TVDB-wrapped client
type Client struct{}

// NewClient returns a new imdb client
func NewClient() *Client {
	return &Client{}
}

// GetSeries returns a series by ID
func (c *Client) GetSeries(mediaID string, mediaType api.Media_MediaType, strid string) (providerapi.Series, error) {
	a, err := kitsu.GetAnime(strid)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to get kitsu anime")
	}

	rating, err := strconv.ParseFloat(a.Attributes.AverageRating, 32)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to parse ratings as int")
	}

	var firstAired time.Time
	if a.Attributes.StartDate != "" {
		time, err := time.Parse("2006-01-02", a.Attributes.StartDate)
		if err != nil {
			return providerapi.Series{}, errors.Wrap(err, "failed to parse startDate as time")
		}

		firstAired = time
	}

	// TODO(jaredallard): find non finished status
	var status providerapi.SeriesStatus
	switch a.Attributes.Status {
	case "finished":
		status = providerapi.SeriesEnded
		break
	}

	images := make([]providerapi.Image, 2)
	images[0] = providerapi.Image{
		Type:   providerapi.ImagePoster,
		URL:    a.Attributes.PosterImage.Original,
		Rating: 10,
	}

	images[1] = providerapi.Image{
		Type:   providerapi.ImageBackground,
		URL:    a.Attributes.CoverImage.Original,
		Rating: 10,
	}

	s := providerapi.Series{
		ID:         mediaID,
		Type:       mediaType,
		Title:      a.Attributes.CanonicalTitle,
		Provider:   Provider,
		ProviderID: strid,
		Overview:   a.Attributes.Synopsis,

		// 100 -> 10
		Rating: float32(rating * 0.10),

		FirstAired: firstAired,
		Status:     status,

		// These aren't exposed by Kitsu
		Network:      "",
		Genre:        []string{},
		Airs:         "",
		AirDayOfWeek: "",
		Runtime:      0,

		Images: images,
	}
	return s, nil
}

// GetEpisodes returns episodes in a series
func (c *Client) GetEpisodes(s *providerapi.Series) ([]providerapi.Episode, error) {
	// there are no episodes in a tv show, but we create a single -1 number episode
	if s.Type == api.Media_MOVIE {
		return []providerapi.Episode{
			providerapi.Episode{
				Number: -1,
			},
		}, nil
	}

	eps, err := kitsu.GetAnimeEpisodes(s.ProviderID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get eps")
	}

	newEps := make([]providerapi.Episode, len(eps))
	for i, e := range eps {
		var firstAired time.Time
		if e.Attributes.Airdate != "" {
			time, err := time.Parse("2006-01-02", e.Attributes.Airdate)
			if err != nil {
				continue
			}

			firstAired = time
		}

		synp := e.Attributes.Synopsis
		if synp == "" {
			synp = "Not Provided"
		}

		newEps[i] = providerapi.Episode{
			Number:       int64(e.Attributes.Number),
			SeasonNumber: int64(e.Attributes.RelativeNumber),
			Season:       e.Attributes.SeasonNumber,
			Name:         e.Attributes.CanonicalTitle,
			Overview:     synp,
			Aired:        firstAired,
			Rating:       10,
		}
	}

	return newEps, nil
}
