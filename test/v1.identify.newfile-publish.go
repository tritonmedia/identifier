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

	i := api.IdentifyNewFile{
		CreatedAt: time.Now().Format(time.RFC3339),
		Quality:   "1080p",
		Key:       "tv/Konosuba/Konosuba S1E2.mkv",
		Episode:   15,
		Season:    1,
		Media: &api.Media{
			Id: "xxx",
		},
	}
	b, err := proto.Marshal(&i)
	if err != nil {
		panic(err)
	}

	if err := client.Publish("v1.identify.newfile", b); err != nil {
		panic(err)
	}

	fmt.Printf("created rmq message, id '%s'", i.Media.Id)
}
