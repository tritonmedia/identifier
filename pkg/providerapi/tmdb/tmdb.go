// Package tmdb is a provider implementation for tmdb support
package tmdb

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	tmdbapi "github.com/ryanbradynd05/go-tmdb"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

const (
	// Provider is the provider enum entry for this provider
	Provider = int(api.Media_TMDB)

	// ProviderImageEndpoint is the endpoint to get an image
	// You can get this from /3/configuration
	ProviderImageEndpoint = "https://image.tmdb.org/t/p/original"
)

// image is an abstraction over movie/tv images
type image struct {
	FilePath string
	Type     providerapi.ImageType
	Height   int
	Width    int
	Rating   float32
}

// Client is a IMDB TVDB-wrapped client
type Client struct {
	t *tmdbapi.TMDb

	// for IMDB provider
	TMDBClient *tmdbapi.TMDb
}

// NewClient returns a new imdb client
func NewClient(apikey string) (*Client, error) {
	config := tmdbapi.Config{
		APIKey:   apikey,
		Proxies:  nil,
		UseProxy: false,
	}

	t := tmdbapi.Init(config)

	return &Client{
		t:          t,
		TMDBClient: t,
	}, nil
}

// getImages returns a list of images for a series
func getImages(images interface{}) ([]providerapi.Image, error) {
	imgs := make([]providerapi.Image, 0)
	switch images.(type) {
	case *tmdbapi.TvImages:
		cast := images.(*tmdbapi.TvImages)
		for _, img := range cast.Posters {
			imgs = append(imgs, providerapi.Image{
				URL:        fmt.Sprintf("%s/%s", ProviderImageEndpoint, strings.TrimPrefix(img.FilePath, "/")),
				Type:       providerapi.ImagePoster,
				Resolution: fmt.Sprintf("%dx%d", img.Height, img.Width),
				Rating:     float64(img.VoteAverage),
			})
		}
		for _, img := range cast.Backdrops {
			imgs = append(imgs, providerapi.Image{
				URL:        fmt.Sprintf("%s/%s", ProviderImageEndpoint, strings.TrimPrefix(img.FilePath, "/")),
				Type:       providerapi.ImageBackground,
				Resolution: fmt.Sprintf("%dx%d", img.Height, img.Width),
				Rating:     float64(img.VoteAverage),
			})
		}
		break
	case *tmdbapi.MovieImages:
		cast := images.(*tmdbapi.MovieImages)
		for _, img := range cast.Posters {
			imgs = append(imgs, providerapi.Image{
				URL:        fmt.Sprintf("%s/%s", ProviderImageEndpoint, strings.TrimPrefix(img.FilePath, "/")),
				Type:       providerapi.ImagePoster,
				Resolution: fmt.Sprintf("%dx%d", img.Height, img.Width),
				Rating:     float64(img.VoteAverage),
			})
		}
		for _, img := range cast.Backdrops {
			imgs = append(imgs, providerapi.Image{
				URL:        fmt.Sprintf("%s/%s", ProviderImageEndpoint, strings.TrimPrefix(img.FilePath, "/")),
				Type:       providerapi.ImageBackground,
				Resolution: fmt.Sprintf("%dx%d", img.Height, img.Width),
				Rating:     float64(img.VoteAverage),
			})
		}
		break
	default:
		return nil, fmt.Errorf("failed to detect image type")
	}
	return imgs, nil
}

// GetSeries returns a series by ID
func (c *Client) GetSeries(mediaID string, mediaType api.Media_MediaType, strid string) (providerapi.Series, error) {
	id, err := strconv.Atoi(strid)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to parse id")
	}

	var s providerapi.Series
	switch mediaType {
	case api.Media_MOVIE:
		m, err := c.t.GetMovieInfo(id, map[string]string{})
		if err != nil {
			// try as a tv series
			return providerapi.Series{}, errors.Wrap(err, "failed to find movie series")
		}

		var firstAired time.Time
		if m.ReleaseDate != "" {
			time, err := time.Parse("2006-01-02", m.ReleaseDate)
			if err != nil {
				return providerapi.Series{}, errors.Wrap(err, "failed to parse ReleaseDate as time")
			}

			firstAired = time
		}

		genres := make([]string, len(m.Genres))
		for i, g := range m.Genres {
			genres[i] = g.Name
		}

		imgs, err := c.t.GetMovieImages(id, map[string]string{})
		if err != nil {
			return providerapi.Series{}, errors.Wrap(err, "failed to get movie images")
		}

		provimgs, err := getImages(imgs)
		if err != nil {
			return providerapi.Series{}, err
		}

		s = providerapi.Series{
			Title:      m.Title,
			Type:       mediaType,
			Provider:   Provider,
			ProviderID: strconv.Itoa(id),
			Overview:   m.Overview,
			Rating:     m.VoteAverage,

			// TODO(jaredallard): better handle network selection
			Network:      m.ProductionCompanies[0].Name,
			FirstAired:   firstAired,
			Status:       providerapi.SeriesEnded,
			Genre:        genres,
			Airs:         "00:00",
			AirDayOfWeek: "Sunday",
			Runtime:      int(m.Runtime),
			Images:       provimgs,
		}
		break
	case api.Media_TV:
		t, err := c.t.GetTvInfo(id, map[string]string{})
		if err != nil {
			// try as a tv series
			return providerapi.Series{}, errors.Wrap(err, "failed to find tv series")
		}

		var firstAired time.Time
		if t.FirstAirDate != "" {
			time, err := time.Parse("2006-01-02", t.FirstAirDate)
			if err != nil {
				return providerapi.Series{}, errors.Wrap(err, "failed to parse firstAirDate as time")
			}

			firstAired = time
		}

		var status providerapi.SeriesStatus
		switch t.Status {
		case "Ended":
			status = providerapi.SeriesEnded
			break
		}

		genres := make([]string, len(t.Genres))
		for i, g := range t.Genres {
			genres[i] = g.Name
		}

		imgs, err := c.t.GetTvImages(id, map[string]string{})
		if err != nil {
			return providerapi.Series{}, errors.Wrap(err, "failed to get tv images")
		}

		provimgs, err := getImages(imgs)
		if err != nil {
			return providerapi.Series{}, err
		}

		s = providerapi.Series{
			Title:      t.Name,
			Type:       mediaType,
			Provider:   Provider,
			ProviderID: strconv.Itoa(id),
			Overview:   t.Overview,
			Rating:     t.VoteAverage,

			// TODO(jaredallard): better handle network selection
			Network:      t.Networks[0].Name,
			FirstAired:   firstAired,
			Status:       status,
			Genre:        genres,
			Airs:         "00:00",
			AirDayOfWeek: "Sunday",
			Runtime:      t.EpisodeRunTime[0],
			Images:       provimgs,
		}
		break
	default:
		return providerapi.Series{}, fmt.Errorf("unsupported media type %s", mediaType.String())
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

	id, err := strconv.Atoi(s.ProviderID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse id")
	}

	t, err := c.t.GetTvInfo(id, map[string]string{})
	if err != nil {
		// try as a tv series
		return nil, errors.Wrap(err, "failed to find tv series")
	}

	// TODO(jaredallard): handle rate limits
	eps := make([]providerapi.Episode, 0)
	numOfEps := 0
	for _, season := range t.Seasons {
		s, err := c.t.GetTvSeasonInfo(id, season.SeasonNumber, map[string]string{})
		if err != nil {
			return nil, fmt.Errorf("failed to read episodes: %v", err)
		}

		for _, e := range s.Episodes {
			var firstAired time.Time
			if e.AirDate != "" {
				time, err := time.Parse("2006-01-02", e.AirDate)
				if err != nil {
					return nil, errors.Wrap(err, "failed to parse airDate as time")
				}

				firstAired = time
			}

			// no absolute episode number support, we just assume its +1
			numOfEps++

			eps = append(eps, providerapi.Episode{
				Number:       int64(numOfEps),
				SeasonNumber: int64(e.EpisodeNumber),
				Season:       season.SeasonNumber, // FIXME
				Name:         e.Name,
				Overview:     e.Overview,
				Aired:        firstAired,
				Rating:       e.VoteAverage,
				Thumb: providerapi.Image{
					Type:       providerapi.ImageThumbnail,
					Resolution: "",
					URL:        fmt.Sprintf("%s/%s", ProviderImageEndpoint, strings.TrimPrefix(e.StillPath, "/")),
					Rating:     10,
				},
			})
		}
	}

	return eps, nil
}
