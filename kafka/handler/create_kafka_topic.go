package handler

import (
	"github.com/pkg/errors"
	"gitlab.id.vin/vincart/golib-message-bus/kafka/core"
	"gitlab.id.vin/vincart/golib-message-bus/kafka/properties"
)

func CreateKafkaTopicHandler(admin core.Admin, props *properties.TopicAdmin) error {
	err := admin.CreateTopics(props.Topics)
	if err != nil {
		return errors.WithMessage(err, "failed create topics")
	}
	return nil
}
