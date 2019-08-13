// Package postgres implements a postgres storageapi interface
package postgres

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"time"

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

	// check if we need to init
	// TODO(jaredallard): we need a migration system and stuff here
	if _, err := cli.Query("SELECT id FROM episodes_v1 LIMIT 1;"); err != nil {
		logrus.Infof("running '%s'", s)
		if _, err := cli.Exec(s); err != nil {
			return nil, errors.Wrap(err, "failed to init database")
		}
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

		var aired string
		if !e.Aired.IsZero() {
			aired = e.Aired.Format(time.RFC3339)
		} else { // default to now
			aired = time.Now().Format(time.RFC3339)
		}

		logrus.Infof("inserting episode '%s': number=%d,air_date='%s'", id.String(), e.Number, aired)
		tx.Exec(`
			INSERT INTO episodes_v1 
				(id, media_id, absolute_number, season, season_number, description, air_date)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id.String(), s.ID, e.Number, e.Season, e.SeasonNumber, e.Overview, aired)
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
