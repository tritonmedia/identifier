// Package imdb is a wrapper around tvdb's support for imdb ids
package imdb

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	"github.com/tritonmedia/identifier/pkg/providerapi/tmdb"
	"github.com/tritonmedia/identifier/pkg/providerapi/tvdb"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

const (
	// Provider is the provider enum entry for this provider
	Provider = int(api.Media_IMDB)
)

// Client is a IMDB TVDB-wrapped client
type Client struct {
	tvdb *tvdb.Client
	tmdb *tmdb.Client
}

// NewClient returns a new imdb client
func NewClient(t *tvdb.Client, tmdb *tmdb.Client) *Client {
	return &Client{
		tmdb: tmdb,
		tvdb: t,
	}
}

// GetSeries returns a series by ID
func (c *Client) GetSeries(mediaID string, mediaType api.Media_MediaType, strid string) (providerapi.Series, error) {
	switch mediaType {
	case api.Media_TV:
		log.Infoln("IMDB provider is using TVDB for media lookup")
		series, err := c.tvdb.TVDBclient.SearchByImdbID(strid)
		if err != nil {
			return providerapi.Series{}, errors.Wrap(err, "failed to search by imdb id via TVDB")
		}

		if len(series) != 1 {
			return providerapi.Series{}, errors.Wrap(err, "found multiple results for this imdb id")
		}
		return c.tvdb.GetSeries(mediaID, mediaType, strconv.Itoa(series[0].ID))
	case api.Media_MOVIE:
		log.Infoln("IMDB provider is using TMDB for media lookup")
		found, err := c.tmdb.TMDBClient.GetFind(strid, "imdb_id", map[string]string{})
		if err != nil {
			return providerapi.Series{}, errors.Wrap(err, "failed to search by imdb id via TMDB")
		}

		if len(found.MovieResults) != 1 {
			return providerapi.Series{}, errors.Wrap(err, "found multiple results, or no results, for this imdb id")
		}
		return c.tmdb.GetSeries(mediaID, mediaType, strconv.Itoa(found.MovieResults[0].ID))
	default:
		return providerapi.Series{}, fmt.Errorf("failed to get series: unknown media type %s", mediaType.String())
	}
}

// GetEpisodes returns episodes in a series
func (c *Client) GetEpisodes(s *providerapi.Series) ([]providerapi.Episode, error) {
	if s.Type == api.Media_TV {
		return c.tvdb.GetEpisodes(s)
	} else if s.Type == api.Media_MOVIE {
		return c.tmdb.GetEpisodes(s)
	} else {
		return nil, fmt.Errorf("failed to get episodes: unknown media type %s", s.Type.String())
	}
}
