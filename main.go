package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/minio/minio-go"
	"github.com/oz/osdb"
	log "github.com/sirupsen/logrus"
	"github.com/tritonmedia/identifier/events"
	"github.com/tritonmedia/identifier/pkg/image"
	"github.com/tritonmedia/identifier/pkg/providerapi"
	"github.com/tritonmedia/identifier/pkg/providerapi/imdb"
	"github.com/tritonmedia/identifier/pkg/providerapi/kitsu"
	"github.com/tritonmedia/identifier/pkg/providerapi/tmdb"
	"github.com/tritonmedia/identifier/pkg/providerapi/tvdb"
	"github.com/tritonmedia/identifier/pkg/rabbitmq"
	"github.com/tritonmedia/identifier/pkg/storageapi/postgres"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	if os.Getenv("IDENTIFIER_DEBUG") != "" {
		log.SetReportCaller(true)
	}

	enabledProviders := []api.Media_MetadataType{api.Media_TVDB, api.Media_TMDB, api.Media_IMDB, api.Media_KITSU}

	providers := make(map[api.Media_MetadataType]providerapi.Fetcher)

	for _, p := range enabledProviders {
		envBase := fmt.Sprintf("IDENTIFIER_%s", strings.ToUpper(p.String()))

		var provider providerapi.Fetcher
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
				log.Errorf("failed to load tvdb provider: %v", err)
				continue
			}

			provider = prov
			break
		case api.Media_IMDB:
			if providers[api.Media_TVDB] == nil {
				log.Errorf("IMDB api wraps TVDB, and TVDB wasn't loaded, refusing to load")
				continue
			}
			if providers[api.Media_TMDB] == nil {
				log.Errorf("IMDB api wraps TMDB, and TMDB wasn't loaded, refusing to load")
				continue
			}

			t := providers[api.Media_TVDB].(*tvdb.Client)
			tmdb := providers[api.Media_TMDB].(*tmdb.Client)

			provider = imdb.NewClient(t, tmdb)
		case api.Media_KITSU:
			provider = kitsu.NewClient()
		case api.Media_TMDB:
			apiKey := os.Getenv(fmt.Sprintf("%s_APIKEY", envBase))

			prov, err := tmdb.NewClient(apiKey)
			if err != nil {
				log.Errorf("failed to load tmdb provider: %v", err)
				continue
			}

			provider = prov
		default:
			log.Errorf("unknown media provider id %d (%s)", p, p.String())
		}

		providers[p] = provider
	} // for loop end

	amqpEndpoint := os.Getenv("IDENTIFIER_RABBITMQ_ENDPOINT")
	if amqpEndpoint == "" {
		amqpEndpoint = "amqp://user:bitnami@127.0.0.1:5672"
		log.Warnf("IDENTIFIER_RABBITMQ_ENDPOINT not defined, defaulting to local config: %s", amqpEndpoint)
	}

	log.Infoln("connecting to rabbitmq ...")
	client, err := rabbitmq.NewClient(amqpEndpoint)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	log.Infoln("connecting to postgres ...")
	db, err := postgres.NewClient()
	if err != nil {
		log.Fatalf("failed to initialize postgres: %v", err)
	}

	b := "IDENTIFIER_S3_"

	var ssl bool
	endpoint := os.Getenv(b + "ENDPOINT")
	u, err := url.Parse(endpoint)
	if err != nil {
		log.Fatalf("failed to parse minio endpoint as a URL: %v", err)
	}

	if u.Scheme == "https" {
		log.Infof("minio: enabled TLS")
		ssl = true
	}

	m, err := minio.New(
		u.Host,
		os.Getenv(b+"ACCESS_KEY"),
		os.Getenv(b+"SECRET_KEY"),
		ssl,
	)
	if err != nil {
		log.Fatalf("failed to create minio (s3) client: %v", err)
	}

	if _, err := m.ListBuckets(); err != nil {
		log.Fatalf("failed to test s3 authentication: %v", err)
	}

	if err := m.MakeBucket("triton-media", "us-west-2"); err != nil {
		log.Warnf("failed to make bucket: %v", err)
	}
	log.Infoln("connected to s3-compatible storage")

	oc, err := osdb.NewClient()
	if err != nil {
		log.Fatalf("failed to create osdb client: %v", err)
	}

	oc.UserAgent = "TemporaryUserAgent"

	// TODO(jaredallard): support for multiple different languages
	if err := oc.LogIn(os.Getenv("OSDB_USERNAME"), os.Getenv("OSDB_PASSWORD"), "eng"); err != nil {
		log.Fatalf("failed to login to opensubtitles: %v", err)
	}

	log.Infoln("connected to opensubtitles (osdb)")

	imageDownloader := image.NewDownloader()
	imageUploader := image.NewUploader(m, "triton-media")

	conf := &events.ProcessorConfig{
		Providers:       providers,
		DB:              db,
		ImageDownloader: imageDownloader,
		ImageUploader:   imageUploader,
		OSDB:            oc,
		S3Client:        m,
	}
	v1identify := events.NewV1IdentifyProcessor(conf)
	v1identifynewfile := events.NewV1IdentifyNewFileProcessor(conf)

	// we are pretty network dependant and slow, so only process 5 at a time
	client.SetPrefetch(5)

	// TODO(jaredallard): pass stop chan
	go func() {
		msgs, err := client.Consume("v1.identify")
		if err != nil {
			log.Fatalf("failed to consume from queues: %v", err)
		}

		for msg := range msgs {
			go v1identify.Process(msg)
		}
	}()

	log.WithField("event", "started").Infoln("waiting for rabbitmq messages")

	msgs, err := client.Consume("v1.identify.newfile")
	if err != nil {
		log.Fatalf("failed to consume from queues: %v", err)
	}

	for msg := range msgs {
		go v1identifynewfile.Process(msg)
	}
}
