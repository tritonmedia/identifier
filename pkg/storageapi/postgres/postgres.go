// Package postgres implements a postgres storageapi interface
package postgres

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/tritonmedia/identifier/pkg/providerapi"
	assets "github.com/tritonmedia/identifier/pkg/storageapi/postgres/schema"
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

	cli, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     "127.0.0.1",
			User:     "postgres",
			Database: "media",
		},
	})
	if err != nil {
		return nil, err
	}

	c := &Client{
		sql: cli,
	}

	// TODO(jaredallard): give up eventually
	err = backoff.Retry(func() error {
		// check if we need to init
		// TODO(jaredallard): we need a migration system and stuff here
		if _, err := cli.Query("SELECT id FROM episodes_v1 LIMIT 1;"); err != nil {
			logrus.Infof("running '%s'", s)
			if _, err := cli.Exec(s); err != nil {
				return errors.Wrap(err, "failed to init database")
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres after substantial retries")
	}

	return c, nil
}

// NewSeries creates a new series
// TODO(jaredallard): this will be added if we ever use this for media population
func (c *Client) NewSeries(s *providerapi.Series) error {
	return nil
}

// NewEpisodes inserts a new episodes
func (c *Client) NewEpisodes(s *providerapi.Series, eps []providerapi.Episode) error {
	tx, err := c.sql.Begin()
	if err != nil {
		return err
	}

	for _, e := range eps {
		id, err := uuid.NewV4()
		if err != nil {
			return errors.Wrap(err, "failed to generate id for episode")
		}

		e.ID = id.String()

		var aired string
		if !e.Aired.IsZero() {
			aired = e.Aired.Format(time.RFC3339)
		} else { // default to now
			aired = time.Now().Format(time.RFC3339)
		}

		logrus.Infof("inserting episode '%s': number=%d,air_date='%s'", id.String(), e.Number, aired)
		if _, err := tx.Exec(`
			INSERT INTO episodes_v1 
				(id, media_id, absolute_number, season, season_number, description, air_date)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id.String(), s.ID, e.Number, e.Season, e.SeasonNumber, e.Overview, aired); err != nil {
			return errors.Wrap(err, "failed to add episode")
		}
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

	r.Next()
	vals, err := r.Values()
	if err != nil {
		return "", err
	}

	if len(vals) != 1 {
		return "", fmt.Errorf("failed to find an episode matching your criteria")
	}

	return vals[0].(string), err
}
