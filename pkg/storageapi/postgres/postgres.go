// Package postgres implements a postgres storageapi interface
package postgres

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gofrs/uuid/v3"
	"github.com/jackc/pgx/v4"
	"github.com/oz/osdb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/tritonmedia/identifier/pkg/providerapi"
	assets "github.com/tritonmedia/identifier/pkg/storageapi/postgres/schema"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

// Client is a postgres client
type Client struct {
	sql *pgx.ConnPool
}

// NewClient returns a new storageapi compatible database provider
func NewClient() (*Client, error) {
	b, _, _, err := assets.Asset("", "/schema.sql")
	if err != nil {
		return nil, err
	}

	// apparently they are gzipped, idk how i feel about that
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode schema from binary")
	}

	b, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode schema from binary")
	}

	s := string(b)

	var conn *pgx.ConnPool

	pgEndpoint := os.Getenv("POSTGRES_ENDPOINT")
	if pgEndpoint == "" {
		pgEndpoint = "127.0.0.1"
		log.Warnf("POSTGRES_ENDPOINT not defined, defaulting to local config: %s", pgEndpoint)
	}

	// TODO(jaredallard): give up eventually
	err = backoff.Retry(func() error {
		var err error
		conn, err = pgx.NewConnPool(pgx.ConnPoolConfig{
			ConnConfig: pgx.ConnConfig{
				Host:     pgEndpoint,
				User:     "postgres",
				Password: os.Getenv("POSTGRES_PASSWORD"),
				Database: "media",
			},
		})
		if err != nil {
			log.Errorf("failed to connect to postgres: %v", err)
		}
		return err
	}, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres after substantial retries")
	}

	// check if we need to init
	// TODO(jaredallard): we need a migration system and stuff here
	if _, err := conn.Query("SELECT id FROM episodes_v1 LIMIT 1;"); err != nil {
		log.Infof("running '%s'", s)
		if _, err := conn.Exec(s); err != nil {
			return nil, errors.Wrap(err, "failed to init database")
		}
	}

	c := &Client{
		sql: conn,
	}

	return c, nil
}

// NewSeries creates a new series
func (c *Client) NewSeries(s *providerapi.Series) error {
	log.Infof("creating series '%s': %v", s)
	if _, err := c.sql.Exec(`
		INSERT INTO series_v1
			(id, title, type, rating, overview, network, first_aired, status, genres, airs, air_day_of_week, runtime)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, s.ID, s.Title, s.Type, s.Rating, s.Overview, s.Network, s.FirstAired, s.Status, strings.Join(s.Genre, ","), s.Airs, s.AirDayOfWeek, s.Runtime); err != nil {
		return errors.Wrap(err, "failed to create series")
	}

	return nil
}

// NewEpisodes inserts a new episodes
func (c *Client) NewEpisodes(s *providerapi.Series, eps []providerapi.Episode) error {
	tx, err := c.sql.Begin()
	if err != nil {
		return err
	}

	for i, e := range eps {
		id, err := uuid.NewV4()
		if err != nil {
			return errors.Wrap(err, "failed to generate id for episode")
		}

		e.ID = id.String()

		// we're a movie, so modify it a bit
		if e.Number == -1 && len(eps) == 1 {
			e.Name = s.Title
		}

		var aired string
		if !e.Aired.IsZero() {
			aired = e.Aired.Format(time.RFC3339)
		} else { // default to now
			aired = time.Now().Format(time.RFC3339)
		}

		log.Infof("inserting episode '%s': season=%d,number=%d,season_number=%d,air_date='%s'", id.String(), e.Season, e.Number, e.SeasonNumber, aired)
		if _, err := tx.Exec(`
			INSERT INTO episodes_v1 
				(id, media_id, absolute_number, season, season_number, description, title, air_date)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, id.String(), s.ID, e.Number, e.Season, e.SeasonNumber, e.Overview, e.Name, aired); err != nil {
			return errors.Wrap(err, "failed to add episode")
		}

		// reflect our changes
		eps[i] = e
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to add episodes")
	}

	return nil
}

// NewImage adds a new image and returns the image ID
func (c *Client) NewImage(s *providerapi.Series, i *providerapi.Image) (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate id for image")
	}

	_, err = c.sql.Exec(`
		INSERT INTO images_v1
			(id, media_id, image_type, checksum, rating, resolution)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id.String(), s.ID, i.Type, i.Checksum, i.Rating, i.Resolution)

	return id.String(), err
}

// NewEpisodeImage adds a new episode image and returns the image ID
func (c *Client) NewEpisodeImage(e *providerapi.Episode, i *providerapi.Image) (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate id for image")
	}

	_, err = c.sql.Exec(`
		INSERT INTO episode_images_v1
			(id, episode_id, image_type, checksum, rating, resolution)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id.String(), e.ID, i.Type, i.Checksum, i.Rating, i.Resolution)

	return id.String(), err
}

// NewEpisodeFile adds a new episode file
func (c *Client) NewEpisodeFile(e *providerapi.Episode, key, quality string) (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate id for episode file")
	}

	_, err = c.sql.Exec(`
		INSERT INTO episode_files_v1
			(id, episode_id, key, quality)
		VALUES ($1, $2, $3, $4)
	`, id.String(), e.ID, key, quality)

	return id.String(), err
}

// FindEpisodeID returns an episode id from episode and season.
func (c *Client) FindEpisodeID(mediaID string, episode, season int) (string, error) {
	r, err := c.sql.Query(`
		SELECT id FROM episodes_v1 WHERE season_number = $1 AND season = $2 AND media_id = $3 LIMIT 1
	`, episode, season, mediaID)
	if err != nil {
		return "", errors.Wrap(err, "failed to search for episode id")
	}
	defer r.Close()

	if !r.Next() {
		return "", fmt.Errorf("failed to find an episode matching your criteria")
	}

	vals, err := r.Values()
	if err != nil {
		return "", err
	}

	if len(vals) != 1 {
		return "", fmt.Errorf("failed to find an episode matching your criteria")
	}

	return vals[0].(string), err
}

// GetSeriesByID returns a series by ID
func (c *Client) GetSeriesByID(mediaID string) (providerapi.Series, error) {
	r, err := c.sql.Query(`
		SELECT 
			id, title, type, rating, overview,
			network, first_aired, status, genres, 
			airs, air_day_of_week, runtime
		FROM series_v1 WHERE id = $1 LIMIT 1
	`, mediaID)
	if err != nil {
		return providerapi.Series{}, errors.Wrap(err, "failed to search for series id")
	}
	defer r.Close()

	if !r.Next() {
		return providerapi.Series{}, fmt.Errorf("failed to find a series matching provided id")
	}

	var id string
	var title string
	var mediaType api.Media_MediaType
	var rating float32
	var overview string
	var network string
	var firstAired time.Time
	var status providerapi.SeriesStatus
	var genres string
	var airs string
	var airDayOfWeek string
	var runtime int

	if err := r.Scan(
		&id, &title, &mediaType, &rating,
		&overview, &network, &firstAired, &status,
		&genres, &airs, &airDayOfWeek, &runtime,
	); err != nil {
		return providerapi.Series{}, err
	}

	return providerapi.Series{
		ID:           id,
		Title:        title,
		Type:         mediaType,
		Rating:       rating,
		Overview:     overview,
		Network:      network,
		FirstAired:   firstAired,
		Status:       status,
		Genre:        strings.Split(genres, ","),
		Airs:         airs,
		AirDayOfWeek: airDayOfWeek,
		Runtime:      runtime,
	}, err
}

// GetEpisodeByID returns an episode by ID
func (c *Client) GetEpisodeByID(s *providerapi.Series, episodeID string) (providerapi.Episode, error) {
	r, err := c.sql.Query(`
		SELECT 
			absolute_number, description, title,
			season, season_number, air_date
		FROM episodes_v1 WHERE id = $1 AND media_id = $2 LIMIT 1
	`, episodeID, s.ID)
	if err != nil {
		return providerapi.Episode{}, errors.Wrap(err, "failed to search for episode id")
	}
	defer r.Close()

	if !r.Next() {
		return providerapi.Episode{}, fmt.Errorf("failed to find episode with that id")
	}

	var absoluteNumber int64
	var overview string
	var title string
	var season int
	var seasonNumber int64
	var airDate time.Time

	if err := r.Scan(
		&absoluteNumber, &overview, &title,
		&season, &seasonNumber, &airDate,
	); err != nil {
		return providerapi.Episode{}, err
	}

	return providerapi.Episode{
		ID:           episodeID,
		Number:       absoluteNumber,
		Overview:     overview,
		Name:         title,
		Season:       season,
		SeasonNumber: seasonNumber,
		Aired:        airDate,
	}, err
}

// NewSubtitle creates a new subtitle
// returns subtitle id and key
// TODO(jaredallard): create providerapi.Subtitle
func (c *Client) NewSubtitle(s *providerapi.Series, e *providerapi.Episode, sub *osdb.Subtitle) (string, string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", "", errors.Wrap(err, "failed to generate id for episode file")
	}

	key := fmt.Sprintf("subtitles/%s/%s/%s.vtt", s.ID, e.ID, id.String())

	_, err = c.sql.Exec(`
		INSERT INTO subtitles_v1
			(id, episode_id, key, language)
		VALUES ($1, $2, $3, $4)
	`, id.String(), e.ID, key, sub.SubLanguageID)

	return id.String(), key, err
}
