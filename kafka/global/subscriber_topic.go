package global

import (
	"fmt"
	"strings"
	"sync"

	"github.com/golibs-starter/golib-message-bus/kafka/core"
	"github.com/golibs-starter/golib-message-bus/kafka/properties"
)

type SubscriberTopicHandler func(message *core.ConsumerMessage)

// Add new type for close handler
type SubscriberCloseHandler func()

// Add new options type
type SubscriberOptions struct {
	CloseHandler SubscriberCloseHandler
}

type SubscriberData struct {
	properties.TopicConsumer
	Handler core.ConsumerHandler
}

type SubscriberTopic struct {
	mapping map[string]SubscriberData

	handlers []core.ConsumerHandler
	mu       sync.RWMutex
}

func NewSubscriberTopic() *SubscriberTopic {
	return &SubscriberTopic{
		mapping:  make(map[string]SubscriberData),
		handlers: []core.ConsumerHandler{},
	}
}

func (s *SubscriberTopic) Register(topic string, handler SubscriberTopicHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	consumerHandler := s.createConsumerHandler(topic, handler)
	s.mapping[topic] = SubscriberData{
		TopicConsumer: properties.TopicConsumer{
			Enable:  true,
			Topic:   topic,
			GroupId: fmt.Sprintf("group-handler-%s-%s", topic, "default"),
		},
		Handler: consumerHandler,
	}
	s.handlers = append(s.handlers, consumerHandler)
}

// RegisterWithOption is used to register a handler with some options
func (s *SubscriberTopic) RegisterWithOption(topic string, handler SubscriberTopicHandler, options properties.TopicConsumer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	consumerHandler := s.createConsumerHandler(topic, handler)
	s.mapping[topic] = SubscriberData{
		TopicConsumer: options,
		Handler:       consumerHandler,
	}
	s.handlers = append(s.handlers, consumerHandler)
}

func (s *SubscriberTopic) GetHandler(topic string) core.ConsumerHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	topic = strings.ToLower(strings.TrimSpace(topic))
	return s.mapping[topic].Handler
}

// get map of topic consumer
func (s *SubscriberTopic) GetTopicConsumerMap() map[string]properties.TopicConsumer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]properties.TopicConsumer)
	for topic, data := range s.mapping {
		if !data.Enable {
			continue
		}
		result[topic] = data.TopicConsumer
	}
	return result
}

// GetHandlers returns all registered handlers
func (s *SubscriberTopic) GetHandlers() []core.ConsumerHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.handlers
}

// create a new consumer handler
func (s *SubscriberTopic) createConsumerHandler(topic string, handler SubscriberTopicHandler) core.ConsumerHandler {
	return &subscriberConsumerHandler{
		handler:      handler,
		closeHandler: nil,
		name:         topic,
	}
}

// subscriberConsumerHandler implements core.ConsumerHandler
type subscriberConsumerHandler struct {
	handler      SubscriberTopicHandler
	closeHandler SubscriberCloseHandler
	name         string
}

// HandlerFunc implements core.ConsumerHandler interface
func (h *subscriberConsumerHandler) HandlerFunc(message *core.ConsumerMessage) {
	h.handler(message)
}

// Close implements core.ConsumerHandler interface
func (h *subscriberConsumerHandler) Close() {
	if h.closeHandler != nil {
		h.closeHandler()
	}
}

// Name returns the handler name for reflection
func (h *subscriberConsumerHandler) Name() string {
	return h.name
}

// Add new method to create handler with options
func (s *SubscriberTopic) RegisterWithClose(topic string, handler SubscriberTopicHandler, closeHandler SubscriberCloseHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	consumerHandler := s.createConsumerHandlerWithClose(topic, handler, closeHandler)
	s.mapping[topic] = SubscriberData{
		TopicConsumer: properties.TopicConsumer{
			Enable:  true,
			Topic:   topic,
			GroupId: fmt.Sprintf("group-handler-%s-%s", topic, "default"),
		},
		Handler: consumerHandler,
	}
	s.handlers = append(s.handlers, consumerHandler)
}

// Add helper method to create handler with close handler
func (s *SubscriberTopic) createConsumerHandlerWithClose(topic string, handler SubscriberTopicHandler, closeHandler SubscriberCloseHandler) core.ConsumerHandler {
	return &subscriberConsumerHandler{
		handler:      handler,
		closeHandler: closeHandler,
		name:         topic,
	}
}

var SubscriberTopicInstance *SubscriberTopic

func init() {
	once := sync.Once{}
	once.Do(func() {
		SubscriberTopicInstance = NewSubscriberTopic()
	})
}
