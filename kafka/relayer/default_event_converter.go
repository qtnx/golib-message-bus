package relayer

import (
	"encoding/json"
	"github.com/pkg/errors"
	kafkaConstant "gitlab.com/golibs-starter/golib-message-bus/kafka/constant"
	"gitlab.com/golibs-starter/golib-message-bus/kafka/core"
	"gitlab.com/golibs-starter/golib-message-bus/kafka/properties"
	"gitlab.com/golibs-starter/golib/config"
	"gitlab.com/golibs-starter/golib/pubsub"
	"gitlab.com/golibs-starter/golib/web/constant"
	webEvent "gitlab.com/golibs-starter/golib/web/event"
	webLog "gitlab.com/golibs-starter/golib/web/log"
	"strings"
)

type DefaultEventConverter struct {
	appProps           *config.AppProperties
	eventProducerProps *properties.EventProducer
}

func NewDefaultEventConverter(
	appProps *config.AppProperties,
	eventProducerProps *properties.EventProducer,
) *DefaultEventConverter {
	return &DefaultEventConverter{
		appProps:           appProps,
		eventProducerProps: eventProducerProps,
	}
}

func (d DefaultEventConverter) Convert(event pubsub.Event) (*core.Message, error) {
	lcEvent := strings.ToLower(event.Name())
	eventTopic := d.eventProducerProps.EventMappings[lcEvent]
	msgBytes, err := json.Marshal(event)
	if err != nil {
		return nil, errors.WithMessage(err, "marshalling event failed")
	}
	message := core.Message{
		Topic: eventTopic.TopicName,
		Value: msgBytes,
		Headers: []core.MessageHeader{
			{
				Key:   []byte(constant.HeaderEventId),
				Value: []byte(event.Identifier()),
			},
			{
				Key:   []byte(constant.HeaderServiceClientName),
				Value: []byte(d.appProps.Name),
			},
		},
		Metadata: map[string]interface{}{
			kafkaConstant.EventId:   event.Identifier(),
			kafkaConstant.EventName: event.Name(),
		},
	}

	if evtOrderable, ok := event.(EventOrderable); ok {
		message.Key = []byte(evtOrderable.OrderingKey())
	}

	var webAbsEvent *webEvent.AbstractEvent
	if we, ok := event.(webEvent.AbstractEventWrapper); ok {
		webAbsEvent = we.GetAbstractEvent()
		message.Headers = d.appendMsgHeaders(message.Headers, webAbsEvent)
		message.Metadata = d.appendMsgMetadata(message.Metadata.(map[string]interface{}), webAbsEvent)
	}
	return &message, nil
}

func (d DefaultEventConverter) appendMsgHeaders(headers []core.MessageHeader, event *webEvent.AbstractEvent) []core.MessageHeader {
	deviceId, _ := event.AdditionalData[constant.HeaderDeviceId].(string)
	deviceSessionId, _ := event.AdditionalData[constant.HeaderDeviceSessionId].(string)
	clientIpAddress, _ := event.AdditionalData[constant.HeaderClientIpAddress].(string)
	return append(headers,
		core.MessageHeader{
			Key:   []byte(constant.HeaderCorrelationId),
			Value: []byte(event.RequestId),
		},
		core.MessageHeader{
			Key:   []byte(constant.HeaderDeviceId),
			Value: []byte(deviceId),
		},
		core.MessageHeader{
			Key:   []byte(constant.HeaderDeviceSessionId),
			Value: []byte(deviceSessionId),
		},
		core.MessageHeader{
			Key:   []byte(constant.HeaderClientIpAddress),
			Value: []byte(clientIpAddress),
		})
}

func (d DefaultEventConverter) appendMsgMetadata(metadata map[string]interface{}, event *webEvent.AbstractEvent) map[string]interface{} {
	metadata[kafkaConstant.LoggingContext] = webLog.BuildLoggingContextFromEvent(event)
	return metadata
}
