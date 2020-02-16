package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	astisub "github.com/asticode/go-astisub"
	"github.com/golang/protobuf/proto"
	"github.com/minio/minio-go/v6"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	"github.com/tritonmedia/identifier/pkg/rabbitmq"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

// V1IdentifyNewFileProcessor is a v1.identify.newfile processor
type V1IdentifyNewFileProcessor struct {
	config *ProcessorConfig
}

// NewV1IdentifyNewFileProcessor creates a v1.identify.newfile processor
func NewV1IdentifyNewFileProcessor(c *ProcessorConfig) *V1IdentifyNewFileProcessor {
	return &V1IdentifyNewFileProcessor{
		config: c,
	}
}

// downloadSubtitles downloads subtitles for a given episode
// TODO(jaredallard): add support for REST api and mayber other subtitle providers
// TODO(jaredallard): cache query results to use less api calls
func (p *V1IdentifyNewFileProcessor) downloadSubtitles(s *providerapi.Series, e *providerapi.Episode) error {
	query := s.Title
	params := []interface{}{ // why is XML fucking like this
		p.config.OSDB.Token,
		[]interface{}{
			map[string]string{
				"query":   query,
				"episode": strconv.Itoa(int(e.SeasonNumber)),
				"season":  strconv.Itoa(e.Season),
			},
		},
	}

	log.Infof("searching osdb for '%s': episode=%d,season=%d", query, e.SeasonNumber, e.Season)
	subs, err := p.config.OSDB.SearchSubtitles(&params)
	if err != nil && strings.Contains(err.Error(), "429") {
		log.Warnf("handling 429...")
		time.Sleep(5 * time.Second)
		return p.downloadSubtitles(s, e)
	} else if err != nil {
		return errors.Wrapf(err, "failed to search for subtitles with query '%s'", query)
	}

	for _, subtitle := range subs {
		b, err := json.Marshal(subtitle)
		fmt.Println(string(b))

		subtitleID, err := strconv.Atoi(subtitle.IDSubtitleFile)
		if err != nil {
			return errors.Wrap(err, "failed to convert subtitle file id to an int")
		}

		files, err := p.config.OSDB.DownloadSubtitlesByIds([]int{subtitleID})
		if err != nil && strings.Contains(err.Error(), "429") {
			log.Warnf("handling 429...")
			time.Sleep(5 * time.Second)
			return p.downloadSubtitles(s, e)
		} else if err != nil {
			return errors.Wrap(err, "failed to download subtitle")
		}

		if len(files) != 1 {
			return fmt.Errorf("downloaded more than one sub")
		}

		reader, err := files[0].Reader()
		if err != nil {
			return errors.Wrap(err, "failed to get subtitle reader from dl")
		}
		defer reader.Close()

		var subtitleReader io.Reader
		switch subtitle.SubFormat {
		case "vtt":
			subtitleReader = reader
		case "srt":
			log.Infof("converting SRT to VTT")
			buf := &bytes.Buffer{}
			c, err := astisub.ReadFromSRT(reader)
			if err != nil {
				return errors.Wrap(err, "failed to convert from SRT to VTT")
			}
			if err := c.WriteToWebVTT(buf); err != nil {
				return errors.Wrap(err, "failed to convert from SRTto VTT")
			}
			subtitleReader = buf
			break
		case "ass", "ssa":
			log.Infof("converting SSA/ASS to VTT")
			buf := &bytes.Buffer{}
			c, err := astisub.ReadFromSSA(reader)
			if err != nil {
				return errors.Wrap(err, "failed to convert from SSA/ASS to VTT")
			}
			if err := c.WriteToWebVTT(buf); err != nil {
				return errors.Wrap(err, "failed to convert from SSA/ASS to VTT")
			}
			subtitleReader = buf
		default:
			return fmt.Errorf("unsupported subtitle format '%s'", subtitle.SubFormat)
		}

		_, subKey, err := p.config.DB.NewSubtitle(s, e, &subtitle)
		if err != nil {
			return errors.Wrap(err, "failed to create db entry for subtitle")
		}

		// TODO(jaredallard): don't hardcode bucket here
		// subtitles/<media-id>/<episode-id>/<subtitle-id>.<ext>
		if _, err := p.config.S3Client.PutObject("triton-media", subKey, subtitleReader, -1, minio.PutObjectOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// Process processes a AMQP message
func (p *V1IdentifyNewFileProcessor) Process(msg *rabbitmq.Delivery) error {
	var job api.IdentifyNewFile
	if err := proto.Unmarshal(msg.Delivery.Body, &job); err != nil {
		log.WithField("event", "decode-message").Errorf("failed to unmarshal rabbitmq message into protobuf format: %v", err)
		if err := msg.Ack(); err != nil {
			log.Warnf("failed to ack failed message: %v", err)
		}
		return nil
	}

	// stop after 5 retries
	if msg.Metadata.Retries > 5 {
		log.Warnf("skipping message that has failed 5 times: id=%s", job.Media.Id)
		if err := msg.Nack(); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	log.Infof("finding series id '%s'", job.Media.Id)
	s, err := p.config.DB.GetSeriesByID(job.Media.Id)
	if err != nil {
		log.Errorf("failed to find series by id: %v", err)

		if err := msg.Error(); err != nil {
			log.Warnf("failed to ack failed message: %v", err)
		}
		return nil
	}

	log.Infof("registering new file for media '%s': quality=%s key='%s' episode=%d season=%d", job.Media.Id, job.Quality, job.Key, job.Episode, job.Season)
	eID, err := p.config.DB.FindEpisodeID(job.Media.Id, int(job.Episode), int(job.Season))
	if err != nil {
		// TODO(jaredallard): add support for ignoring season number if the metadata provider,
		// such as TVDB, doesn't know of that season for some reason: i.e
		// https://forums.thetvdb.com/viewtopic.php?t=28709
		log.Errorf("failed to find episode id: %v", err)

		if err := msg.Nack(); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	log.Infof("finding episode by id '%s'", eID)
	e, err := p.config.DB.GetEpisodeByID(&s, eID)
	if err != nil {
		log.Errorf("failed to find episode by id: %v", err)
		// TODO(jaredallard): add backoff or something to these
		if err := msg.Error(); err != nil {
			log.Warnf("failed to ack failed message: %v", err)
		}
		return nil
	}

	log.Infof("adding file to episode: id='%s' media_id='%s'", eID, job.Media.Id)
	if _, err := p.config.DB.NewEpisodeFile(&providerapi.Episode{
		ID: eID,
	}, job.Key, job.Quality); err != nil {
		log.Errorf("failed to add episode to the database: %v", err)
		if err := msg.Nack(); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	log.Infof("searching for subtitles for this episode")
	if err := p.downloadSubtitles(&s, &e); err != nil {
		log.Warnf("failed to download subtitles: %v", err)
	}

	// --------
	// ACK
	// --------
	log.Infof("episode file added")
	if err := msg.Ack(); err != nil {
		log.Warnf("failed to ack: %v", err)
	}

	return nil
}
