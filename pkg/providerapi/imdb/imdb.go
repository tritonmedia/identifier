// Package imdb is a wrapper around tvdb's support for imdb ids
package imdb

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/tritonmedia/identifier/pkg/providerapi"
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
}

// NewClient returns a new imdb client
func NewClient(t *tvdb.Client) *Client {
	return &Client{
		tvdb: t,
	}
}

// GetSeries returns a series by ID
func (c *Client) GetSeries(strid string) (providerapi.Series, error) {
	series, err := c.tvdb.TVDBclient.SearchByImdbID(strid)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to search by imdb id")
	}

	if len(series) != 1 {
		return providerapi.Series{}, errors.Wrap(err, "found multiple results for this imdb id")
	}

	tvdbid := strconv.Itoa(series[0].ID)
	return c.tvdb.GetSeries(tvdbid)
}

// GetEpisodes returns episodes in a series
func (c *Client) GetEpisodes(s *providerapi.Series) ([]providerapi.Episode, error) {
	return c.tvdb.GetEpisodes(s)
}
