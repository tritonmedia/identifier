package events

import (
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

// V1IdentifyProcessor process v1.identifiy messages
type V1IdentifyProcessor struct {
	config *ProcessorConfig
}

// NewV1IdentifyProcessor returns and identifier processor
func NewV1IdentifyProcessor(conf *ProcessorConfig) *V1IdentifyProcessor {
	return &V1IdentifyProcessor{
		config: conf,
	}
}

// Process processes an AMQP message
func (p *V1IdentifyProcessor) Process(msg amqp.Delivery) error {
	var job api.Identify
	if err := proto.Unmarshal(msg.Body, &job); err != nil {
		log.WithField("event", "decode-message").Errorf("failed to unmarshal rabbitmq message into protobuf format: %v", err)
		if err := msg.Nack(false, true); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	if job.Media.Id == "" {
		log.Warnf("skipping message due to media.id not being set")
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	log.Infof("identifying media '%s' using provider '%s' with id '%s'", job.Media.Id, job.Media.Metadata.String(), job.Media.MetadataId)

	var prov providerapi.Fetcher
	if prov = p.config.Providers[job.Media.Metadata]; p == nil {
		log.Errorf("provider id '%d' (%s) is not enabled/supported", job.Media.Metadata, job.Media.Metadata.String())
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	s, err := prov.GetSeries(job.Media.Id, job.Media.Type, job.Media.MetadataId)
	if err != nil {
		log.Errorf(err.Error())
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	// TODO(jaredallard): remove when we put series into the database
	if s.ID == "" {
		s.ID = job.Media.Id
	}

	e, err := prov.GetEpisodes(&s)
	if err != nil {
		log.Errorf(err.Error())
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	log.Infof("got '%d' episodes for series '%s'", len(e), s.Title)

	log.Infof("inserting episodes into database")
	if err := p.config.DB.NewEpisodes(&s, e); err != nil {
		log.Errorf("failed to insert: %v", err)
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	// TODO(jaredallard): upload episode images
	log.Info("inserting images into database")
	newImages := make([]providerapi.Image, len(s.Images))
	for i, img := range s.Images {
		log.Infof("downloading image '%v'", img.URL)
		b, err := p.config.ImageDownloader.DownloadImage(&img)
		if err != nil {
			log.Errorf("failed to process image: %v", err)
			return nil
		}

		log.Infof("uploading image '%v'", img.URL)
		id, err := p.config.DB.NewImage(&s, &img)
		if err != nil {
			log.Errorf("failed to add image to the database: %v", err)
			return nil
		}

		if err := p.config.ImageUploader.UploadImage(s.ID, id, b, &img); err != nil {
			log.Errorf("failed to upload image: %v", err)
			return nil
		}
		newImages[i] = img
	}

	// --------
	// ack
	// --------
	if err := msg.Ack(false); err != nil {
		log.Warnf("failed to ack: %v", err)
		return nil // explicit continue here in case anything is added below
	}

	log.Infof("successfully added into the database")
	return nil
}
