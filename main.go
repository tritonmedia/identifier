package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	"github.com/tritonmedia/identifier/pkg/providerapi/imdb"
	"github.com/tritonmedia/identifier/pkg/providerapi/tvdb"
	"github.com/tritonmedia/identifier/pkg/rabbitmq"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	enabledProviders := []api.Media_MetadataType{api.Media_TVDB, api.Media_IMDB}

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
		default:
			log.Errorf("unknown media provider id %d (%s)", p, p.String())
		}

		clients[p] = client
		providers[p] = provider
	}

	client, err := rabbitmq.NewClient("amqp://user:bitnami@127.0.0.1:5672")
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	msgs, err := client.Consume("v1.identify")
	if err != nil {
		log.Fatalf("failed to consume from queues: %v", err)
	}

	log.WithField("event", "started").Infoln("waiting for rabbitmq messages")
	for msg := range msgs {
		var job api.Identify
		if err := proto.Unmarshal(msg.Body, &job); err != nil {
			log.WithField("event", "decode-message").Errorf("failed to unmarshal rabbitmq message into protobuf format: %v", err)
			if err := msg.Nack(false, true); err != nil {
				log.Warnf("failed to nack failed message: %v", err)
			}
			continue
		}

		if job.Media.Id == "" {
			log.Warnf("skipping message due to media.id not being set")
			if err := msg.Nack(false, false); err != nil {
				log.Warnf("failed to nack failed message: %v", err)
			}
			continue
		}

		log.Infof("identifying media '%s' using provider '%s' with id '%s'", job.Media.Id, job.Media.Metadata.String(), job.Media.MetadataId)

		var p providerapi.Fetcher
		if p = providers[job.Media.Metadata]; p == nil {
			log.Errorf("provider id '%d' (%s) is not enabled/supported", job.Media.Metadata, job.Media.Metadata.String())
			if err := msg.Nack(false, false); err != nil {
				log.Warnf("failed to nack failed message: %v", err)
			}
			continue
		}

		s, err := p.GetSeries(job.Media.MetadataId)
		if err != nil {
			log.Errorf(err.Error())
			if err := msg.Nack(false, false); err != nil {
				log.Warnf("failed to nack failed message: %v", err)
			}
			continue
		}

		e, err := p.GetEpisodes(&s)
		if err != nil {
			log.Errorf(err.Error())
			if err := msg.Nack(false, false); err != nil {
				log.Warnf("failed to nack failed message: %v", err)
			}
			continue
		}

		log.Infof("got '%d' episodes for series '%s'", len(e), s.Title)
		if err := msg.Ack(false); err != nil {
			log.Warnf("failed to ack: %v", err)
			continue // explicit continue here in case anything is added below
		}
	}
}
