package handler

import (
	kafkaConstant "gitlab.com/golibs-starter/golib-message-bus/kafka/constant"
	"gitlab.com/golibs-starter/golib-message-bus/kafka/core"
	"gitlab.com/golibs-starter/golib/event"
	"gitlab.com/golibs-starter/golib/web/constant"
	"gitlab.com/golibs-starter/golib/web/log"
	"strings"
)

func ProducerSuccessLogHandler(producer core.AsyncProducer, eventProps *event.Properties) {
	notLogPayloadForEvents := make(map[string]bool)
	for _, e := range eventProps.Log.NotLogPayloadForEvents {
		notLogPayloadForEvents[e] = true
	}
	go func() {
		for msg := range producer.Successes() {
			if metadata, ok := msg.Metadata.(map[string]interface{}); ok {
				eventId, _ := metadata[kafkaConstant.EventId].(string)
				eventName, _ := metadata[kafkaConstant.EventName].(string)
				logContext := []interface{}{constant.ContextReqMeta, getLoggingContext(metadata)}
				if notLogPayloadForEvents[strings.ToLower(eventName)] {
					log.Infow(logContext, "Success to produce Kafka message [%s] with id [%s] to topic [%s]",
						eventName, eventId, msg.Topic)
				} else {
					log.Infow(logContext, "Success to produce Kafka message [%s] with id [%s] to topic [%s], value [%s]",
						eventName, eventId, msg.Topic, string(msg.Value))
				}
			} else {
				log.Infof("Success to produce Kafka message to topic [%s], value [%s]",
					msg.Topic, string(msg.Value))
			}
		}
	}()
}
