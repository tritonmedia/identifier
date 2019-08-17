package events

import (
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tritonmedia/identifier/pkg/providerapi"
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

// Process processes a AMQP message
func (p *V1IdentifyNewFileProcessor) Process(msg amqp.Delivery) error {
	var job api.IdentifyNewFile
	if err := proto.Unmarshal(msg.Body, &job); err != nil {
		log.WithField("event", "decode-message").Errorf("failed to unmarshal rabbitmq message into protobuf format: %v", err)
		if err := msg.Nack(false, true); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	log.Infof("registering new file for media '%s': quality=%s key='%s' episode=%d season=%d", job.Media.Id, job.Quality, job.Key, job.Episode, job.Season)
	eID, err := p.config.DB.FindEpisodeID(job.Media.Id, int(job.Episode), int(job.Season))
	if err != nil {
		log.Errorf("failed to find episode id")
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	log.Infof("adding file to episode: id='%s' media_id='%s'", eID, job.Media.Id)
	if _, err := p.config.DB.NewEpisodeFile(&providerapi.Episode{
		ID: eID,
	}, job.Key, job.Quality); err != nil {
		log.Errorf("failed to add episode to the database: %v", err)
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	// --------
	// ACK
	// --------
	log.Infof("episode file added")
	if err := msg.Ack(false); err != nil {
		log.Warnf("failed to ack: %v", err)
		return nil // explicit continue here in case anything is added below
	}

	return nil
}
