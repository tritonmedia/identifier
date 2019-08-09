package main

import (
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

	msgs, err := client.Consume("v1.identify")
	if err != nil {
		log.Fatalf("failed to consume from queues: %v", err)
	}

	log.WithField("event", "started").Infoln("waiting for rabbitmq messages")
	for msg := range msgs {
		log.Infof("new message '%s'", msg.MessageId)

		var job api.Identify
		if err := proto.Unmarshal(msg.Body, &job); err != nil {
			log.WithField("event", "decode-message").Errorf("failed to unmarshal rabbitmq message into protobuf format: %v", err)
			if err := msg.Nack(false, true); err != nil {
				log.Warnf("failed to nack failed message: %v", err)
			}
			continue
		}

		log.Infof("identifying media '%s' using provider '%s' with id '%s'", job.Media.Id, job.Media.Metadata.String(), job.Media.MetadataId)
	}
}
