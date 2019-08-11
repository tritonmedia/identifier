package main

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/tritonmedia/identifier/pkg/rabbitmq"
	api "github.com/tritonmedia/tritonmedia.go/pkg/proto"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	client, err := rabbitmq.NewClient("amqp://user:bitnami@127.0.0.1:5672")
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	i := api.Identify{
		CreatedAt: time.Now().Format(time.RFC3339),
		Media: &api.Media{
			Id:         "xxx",
			Metadata:   api.Media_TVDB,
			MetadataId: "291627",
		},
	}
	b, err := proto.Marshal(&i)
	if err != nil {
		panic(err)
	}

	if err := client.Publish("v1.identify", b); err != nil {
		panic(err)
	}

	fmt.Printf("created rmq message for metadata '%d' id '%s'", i.Media.Metadata, i.Media.MetadataId)
}
