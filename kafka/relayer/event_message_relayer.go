package relayer

import (
	"encoding/json"
	"strings"
	"sync"

	"fmt"

	"github.com/golibs-starter/golib-message-bus/kafka/core"
	"github.com/golibs-starter/golib-message-bus/kafka/global"
	"github.com/golibs-starter/golib-message-bus/kafka/log"
	"github.com/golibs-starter/golib-message-bus/kafka/properties"
	"github.com/golibs-starter/golib/event"
	coreLog "github.com/golibs-starter/golib/log"
	"github.com/golibs-starter/golib/pubsub"
	"go.uber.org/zap"
)

type HandlerFunc func(message *core.ConsumerMessage) error

type EventMessageRelayer struct {
	producer               core.SyncProducer
	eventProducerProps     *properties.EventProducer
	eventProps             *event.Properties
	notLogPayloadForEvents map[string]bool
	eventConverter         EventConverter
	// Add new fields for callback handling
	handlers   map[string][]HandlerFunc
	handlersMu sync.RWMutex
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
		// Initialize new fields
		handlers: make(map[string][]HandlerFunc),
	}
}

// Supports checks if the given event is supported by the EventMessageRelayer.
// It returns true if the event is supported, otherwise false.
//
// Parameters:
//
//	event (pubsub.Event): The event to be checked.
//
// Returns:
//
//	bool: True if the event is supported, false otherwise.
func (e *EventMessageRelayer) Supports(event pubsub.Event) bool {
	logger := coreLog.WithCtx(event.Context())
	lcEvent := strings.ToLower(event.Name())

	// If the topic is already set in the global mapping, use it
	topic := global.EventTopicMappingInstance.GetTopic(lcEvent)
	if topic != "" {
		return true
	}

	eventTopic, exists := e.eventProducerProps.EventMappings[lcEvent]
	if !exists {
		logger.Warnf("Produce Kafka message is skip, no mapping found for event [%s]", event.Name())
		return false
	}
	if eventTopic.Disable {
		logger.Warnf("Produce Kafka message is disabled for event [%s]", event.Name())
		return false
	}
	if eventTopic.TopicName == "" {
		logger.Errorf("Cannot find topic for event [%s]", event.Name())
		return false
	}
	return true
}

func (e *EventMessageRelayer) Handle(event pubsub.Event) {
	logger := coreLog.WithCtx(event.Context())
	message, err := e.eventConverter.Convert(event)
	if err != nil {
		logger.WithErrors(err).Error("Error while converting event to kafka message")
		return
	}
	partition, offset, err := e.producer.Send(message)
	if err != nil {
		logger.WithErrors(err).Errorf("Error while producing kafka message %s",
			log.DescMessage(message, e.eventProps.Log.NotLogPayloadForEvents))
		return
	}
	logger.Infof("Success to produce to kafka partition [%d], offset [%d], message %s",
		partition, offset, log.DescMessage(message, e.eventProps.Log.NotLogPayloadForEvents))
}

// RegisterHandler registers a callback function for a specific topic
func (e *EventMessageRelayer) RegisterHandler(topicName string, handler any) {
	e.handlersMu.Lock()
	defer e.handlersMu.Unlock()

	if e.handlers[topicName] == nil {
		e.handlers[topicName] = make([]HandlerFunc, 0)
	}
	e.handlers[topicName] = append(e.handlers[topicName], handler.(HandlerFunc))
	coreLog.Infof("Registered new handler for topic [%s]", topicName)
}

// HandleMessage processes a message using registered handlers
func (e *EventMessageRelayer) HandleMessage(message *core.ConsumerMessage) error {
	e.handlersMu.RLock()
	handlers, exists := e.handlers[message.Topic]
	e.handlersMu.RUnlock()

	if !exists {
		return nil // No handlers registered for this topic
	}

	logger := coreLog.WithField(zap.String("topic", message.Topic))

	for _, handler := range handlers {
		if err := handler(message); err != nil {
			logger.WithErrors(err).Error("Handler failed to process message")
			return err
		}
	}

	return nil
}

// Add this helper function instead
func RegisterTypedHandler[T any](relayer pubsub.Subscriber, topicName string, handler func(message T) error) {
	relayer.RegisterHandler(topicName, func(msg *core.ConsumerMessage) error {
		var data T
		if err := json.Unmarshal(msg.Value, &data); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}
		return handler(data)
	})
}
