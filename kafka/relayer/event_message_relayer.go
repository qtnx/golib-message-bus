package relayer

import (
	"gitlab.com/golibs-starter/golib-message-bus/kafka/core"
	"gitlab.com/golibs-starter/golib-message-bus/kafka/log"
	"gitlab.com/golibs-starter/golib-message-bus/kafka/properties"
	"gitlab.com/golibs-starter/golib/event"
	"gitlab.com/golibs-starter/golib/pubsub"
	webLog "gitlab.com/golibs-starter/golib/web/log"
	"strings"
)

type EventMessageRelayer struct {
	producer               core.SyncProducer
	eventProducerProps     *properties.EventProducer
	eventProps             *event.Properties
	notLogPayloadForEvents map[string]bool
	eventConverter         EventConverter
}

func NewEventMessageRelayer(
	producer core.SyncProducer,
	eventProducerProps *properties.EventProducer,
	eventProps *event.Properties,
	eventConverter EventConverter,
) pubsub.Subscriber {
	notLogPayloadForEvents := make(map[string]bool)
	for _, e := range eventProps.Log.NotLogPayloadForEvents {
		notLogPayloadForEvents[e] = true
	}
	return &EventMessageRelayer{
		producer:               producer,
		eventProducerProps:     eventProducerProps,
		eventProps:             eventProps,
		notLogPayloadForEvents: notLogPayloadForEvents,
		eventConverter:         eventConverter,
	}
}

func (e EventMessageRelayer) Supports(event pubsub.Event) bool {
	lcEvent := strings.ToLower(event.Name())
	eventTopic, exists := e.eventProducerProps.EventMappings[lcEvent]
	if !exists {
		webLog.Debuge(event, "Produce Kafka message is skip, no mapping found for event [%s]", event.Name())
		return false
	}
	if eventTopic.Disable {
		webLog.Debuge(event, "Produce Kafka message is disabled for event [%s]", event.Name())
		return false
	}
	if eventTopic.TopicName == "" {
		webLog.Errore(event, "Cannot find topic for event [%s]", event.Name())
		return false
	}
	return true
}

func (e EventMessageRelayer) Handle(event pubsub.Event) {
	message, err := e.eventConverter.Convert(event)
	if err != nil {
		webLog.Errore(event, "Error while converting event to kafka message. Error [%s]", err)
		return
	}
	partition, offset, err := e.producer.Send(message)
	if err != nil {
		webLog.Errore(event, "Error while producing kafka message %s. Error [%s]",
			log.DescMessage(message, e.eventProps.Log.NotLogPayloadForEvents), err)
		return
	}
	webLog.Infoe(event, "Success to produce to kafka partition [%d], offset [%d], message %s",
		partition, offset, log.DescMessage(message, e.eventProps.Log.NotLogPayloadForEvents))
}