package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tritonmedia/identifier/pkg/image"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	"github.com/tritonmedia/identifier/pkg/providerapi/imdb"
	"github.com/tritonmedia/identifier/pkg/providerapi/kitsu"
	"github.com/tritonmedia/identifier/pkg/providerapi/tvdb"
	"github.com/tritonmedia/identifier/pkg/rabbitmq"
	"github.com/tritonmedia/identifier/pkg/storageapi"
	"github.com/tritonmedia/identifier/pkg/storageapi/postgres"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

func processor(providers map[api.Media_MetadataType]providerapi.Fetcher,
	db storageapi.Provider,
	idl *image.Downloader,
	iup *image.Uploader,
	msg amqp.Delivery) error {
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

	var p providerapi.Fetcher
	if p = providers[job.Media.Metadata]; p == nil {
		log.Errorf("provider id '%d' (%s) is not enabled/supported", job.Media.Metadata, job.Media.Metadata.String())
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	s, err := p.GetSeries(job.Media.Id, job.Media.MetadataId)
	if err != nil {
		log.Errorf(err.Error())
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	e, err := p.GetEpisodes(&s)
	if err != nil {
		log.Errorf(err.Error())
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	log.Infof("got '%d' episodes for series '%s'", len(e), s.Title)

	log.Infof("inserting episodes into database")
	if err := db.NewEpisodes(&s, e); err != nil {
		log.Errorf("failed to insert: %v", err)
		if err := msg.Nack(false, false); err != nil {
			log.Warnf("failed to nack failed message: %v", err)
		}
		return nil
	}

	log.Info("inserting images into database")
	newImages := make([]providerapi.Image, len(s.Images))
	for i, img := range s.Images {
		log.Infof("downloading image '%v'", img.URL)
		// TODO(jaredallard): upload image at this step
		b, err := idl.DownloadImage(&img)
		if err != nil {
			log.Errorf("failed to process image: %v", err)
			return nil
		}

		// TODO(jaredallard): set image ID on the actual struct?
		log.Infof("uploading image '%v'", img.URL)
		id, err := db.NewImage(&s, &img)
		if err != nil {
			log.Errorf("failed to add image to the database: %v", err)
			return nil
		}

		if err := iup.UploadImage(s.ID, id, b, &img); err != nil {
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

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	enabledProviders := []api.Media_MetadataType{api.Media_TVDB, api.Media_IMDB, api.Media_KITSU}

	providers := make(map[api.Media_MetadataType]providerapi.Fetcher)
	clients := make(map[api.Media_MetadataType]interface{})

	for _, p := range enabledProviders {
		envBase := fmt.Sprintf("IDENTIFIER_%s", strings.ToUpper(p.String()))

		var provider providerapi.Fetcher
		var client interface{}
		switch p {
		case api.Media_TVDB:
			apiKey := os.Getenv(fmt.Sprintf("%s_APIKEY", envBase))
			userKey := os.Getenv(fmt.Sprintf("%s_USERKEY", envBase))
			username := os.Getenv(fmt.Sprintf("%s_USERNAME", envBase))

			prov, err := tvdb.NewClient(&tvdb.Config{
				APIKey:   apiKey,
				UserKey:  userKey,
				Username: username,
			})
			if err != nil {
				log.Errorf("failed to create tvdb provider: %v", err)
				continue
			}

			client = prov
			provider = prov
			break
		case api.Media_IMDB:
			if clients[api.Media_TVDB] == nil {
				log.Errorf("IMDB api wraps TVDB, and TVDB wasn't loaded, refusing to load")
			}

			t := providers[api.Media_TVDB].(*tvdb.Client)

			provider = imdb.NewClient(t)
		case api.Media_KITSU:
			provider = kitsu.NewClient()
		default:
			log.Errorf("unknown media provider id %d (%s)", p, p.String())
		}

		clients[p] = client
		providers[p] = provider
	} // for loop end

	client, err := rabbitmq.NewClient("amqp://user:bitnami@127.0.0.1:5672")
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	db, err := postgres.NewClient()
	if err != nil {
		log.Fatalf("failed to initialize postgres: %v", err)
	}

	b := "IDENTIFIER_S3_"
	m, err := minio.New(
		os.Getenv(b+"ENDPOINT"),
		os.Getenv(b+"ACCESS_KEY"),
		os.Getenv(b+"SECRET_KEY"),

		// TODO(jaredallard): determine from endpoint url
		false,
	)

	if _, err := m.ListBuckets(); err != nil {
		log.Fatalf("failed to test s3 authentication: %v", err)
	}

	if err := m.MakeBucket("triton-media", "us-west-2"); err != nil {
		log.Warnf("failed to make bucket: %v", err)
	}

	imageDownloader := image.NewDownloader()
	imageUploader := image.NewUploader(m, "triton-media")

	msgs, err := client.Consume("v1.identify")
	if err != nil {
		log.Fatalf("failed to consume from queues: %v", err)
	}

	log.WithField("event", "started").Infoln("waiting for rabbitmq messages")
	for msg := range msgs {
		processor(providers, db, imageDownloader, imageUploader, msg)
	}
}
