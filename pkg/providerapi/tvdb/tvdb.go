// Package tvdb implements a metadata provider
package tvdb

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pioz/tvdb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

// TODO(jaredallard): add JWT refresh logic

const (
	// Provider is the provider enum entry for this provider
	Provider = int(api.Media_TVDB)

	// ProviderImageEndpoint is the image endppint for this provider
	ProviderImageEndpoint = "https://www.thetvdb.com/banners/"
)

// Config is a tvdb config
type Config struct {
	// APIKey is a tvdb api key
	APIKey string

	// UserKey is a user key for tvdb
	UserKey string

	// Username is your tvdb username
	Username string
}

// Client implements the providerapi interface
type Client struct {
	config     *Config
	TVDBclient *tvdb.Client
}

// NewClient returns a tvdb client
func NewClient(c *Config) (*Client, error) {
	client := &tvdb.Client{
		Apikey:   c.APIKey,
		Userkey:  c.UserKey,
		Username: c.Username,
	}

	if err := client.Login(); err != nil {
		return nil, err
	}

	return &Client{
		config:     c,
		TVDBclient: client,
	}, nil
}

func (c *Client) getSeriesImages(seriesID int) ([]providerapi.Image, error) {
	imgs := make([]tvdb.Image, 0)
	types := map[string]func(s *tvdb.Series) error{
		// backgrounds
		"fanart": c.TVDBclient.GetSeriesFanartImages,
		// poster
		"poster": c.TVDBclient.GetSeriesPosterImages,

		// not used yet
		// "series": c.TVDBclient.GetSeriesSeriesImages,
	}

	// hit the api for each type of image we want
	for t, f := range types {
		s := &tvdb.Series{
			ID: seriesID,
		}
		if err := f(s); err != nil {
			return nil, errors.Wrapf(err, "failed to get %s images", t)
		}
		imgs = append(imgs, s.Images...)
	}

	newImgs := make([]providerapi.Image, 0)
	for _, img := range imgs {
		var imageType providerapi.ImageType

		switch img.KeyType {
		case "poster":
			imageType = providerapi.ImagePoster
			break
		case "background":
			imageType = providerapi.ImageBackground
		default: // skip unknown
			continue
		}

		newImgs = append(newImgs, providerapi.Image{
			URL:          fmt.Sprintf("%s%s", ProviderImageEndpoint, img.FileName),
			Rating:       img.RatingsInfo.Average,
			Resolution:   img.Resolution,
			ThumbnailURL: fmt.Sprintf("%s%s", ProviderImageEndpoint, img.Thumbnail),
			Type:         imageType,
		})
	}

	return newImgs, nil
}

// GetEpisodes returns a list of all the episodes in a series
func (c *Client) GetEpisodes(series *providerapi.Series) ([]providerapi.Episode, error) {
	id, err := strconv.Atoi(series.ProviderID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse id")
	}

	s := &tvdb.Series{
		ID: id,
	}
	if err := c.TVDBclient.GetSeriesEpisodes(s, url.Values{}); err != nil {
		return nil, errors.Wrap(err, "failed to get episodes")
	}

	eps := make([]providerapi.Episode, len(s.Episodes))
	for i, e := range s.Episodes {
		img := providerapi.Image{
			Type:       providerapi.ImageThumbnail,
			URL:        fmt.Sprintf("%s%s", ProviderImageEndpoint, e.Filename),
			Rating:     10,
			Resolution: fmt.Sprintf("%sx%s", e.ThumbHeight, e.ThumbWidth),
		}

		var firstAired time.Time
		if e.FirstAired != "" {
			time, err := time.Parse("2006-01-02", e.FirstAired)
			if err != nil {
				logrus.Warnf("failed to parse firstAired for episode")
				continue
			}

			firstAired = time
		}

		eps[i] = providerapi.Episode{
			Number:       int64(e.AbsoluteNumber),
			SeasonNumber: int64(e.AiredEpisodeNumber),
			Season:       e.AiredSeason,
			Name:         e.EpisodeName,
			Overview:     e.Overview,
			Rating:       e.SiteRating,
			Aired:        firstAired,
			Thumb:        img,
		}
	}

	return eps, nil
}

// GetSeries returns a series
func (c *Client) GetSeries(mediaID string, strid string) (providerapi.Series, error) {
	id, err := strconv.Atoi(strid)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to parse id")
	}

	// create the series, populated by the below call
	s := &tvdb.Series{
		ID: id,
	}

	if err := c.TVDBclient.GetSeries(s); err != nil {
		return providerapi.Series{}, err
	}

	imgs, err := c.getSeriesImages(id)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to get images")
	}

	var firstAired time.Time
	if s.FirstAired != "" {
		time, err := time.Parse("2006-01-02", s.FirstAired)
		if err != nil {
			return providerapi.Series{}, errors.Wrap(err, "failed to parse firstAired as time")
		}

		firstAired = time
	}

	var status providerapi.SeriesStatus
	switch s.Status {
	case "Ended":
		status = providerapi.SeriesEnded
		break
	case "Continuing":
		status = providerapi.SeriesAiring
		break
	}

	runtime, err := strconv.Atoi(s.Runtime)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to parse runtime")
	}

	return providerapi.Series{
		ID:           mediaID,
		Title:        s.SeriesName,
		ProviderID:   strid,
		Provider:     Provider,
		Overview:     s.Overview,
		Rating:       s.SiteRating,
		Network:      s.Network,
		FirstAired:   firstAired,
		Status:       status,
		Genre:        s.Genre,
		Airs:         s.AirsTime,
		AirDayOfWeek: s.AirsDayOfWeek,
		Runtime:      runtime,
		Images:       imgs,
	}, nil
}
