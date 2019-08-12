// Package postgres implements a postgres storageapi interface
package postgres

import (
	"bytes"
	"compress/gzip"
	"context"
	"io/ioutil"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/tritonmedia/identifier/pkg/providerapi"
	assets "github.com/tritonmedia/identifier/pkg/storageapi/postgres/schema"
)

// Client is a postgres client
type Client struct {
	sql *pgx.Conn
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

	cli, err := pgx.Connect(pgx.ConnConfig{
		Host:     "127.0.0.1",
		User:     "postgres",
		Database: "media",
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
	b := c.sql.BeginBatch()

	for _, e := range eps {
		id, err := uuid.NewV4()
		if err != nil {
			return errors.Wrap(err, "failed to generate id for episode")
		}

		vals := []interface{}{id.String(), s.ID, e.Number, e.Overview, e.Aired.Format(time.RFC3339)}

		logrus.Infof("inserting episode ID: %s: media_id='%v',number='%v',overview='%v',aired='%v'", vals[0], vals[1], vals[2], vals[3], vals[4])
		b.Queue(`
			INSERT INTO episodes_v1 
				(id, media_id, episode_number, description, air_date)
				VALUES ($1, $2, $3, $4, $5)
		`, vals, []pgtype.OID{pgtype.VarcharOID, pgtype.VarcharOID, pgtype.Int8OID, pgtype.TextOID, pgtype.TimestamptzOID}, nil)
	}

	err := b.Send(context.Background(), nil)
	if err != nil {
		return errors.Wrap(err, "failed to add episodes")
	}
	if _, err := b.ExecResults(); err != nil {
		return errors.Wrap(err, "failed to add episodes")
	}

	return nil
}
