package relayer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	kafkaConstant "github.com/golibs-starter/golib-message-bus/kafka/constant"
	"github.com/golibs-starter/golib-message-bus/kafka/core"
	"github.com/golibs-starter/golib-message-bus/kafka/properties"
	"github.com/golibs-starter/golib/config"
	"github.com/golibs-starter/golib/event"
	"github.com/golibs-starter/golib/web/constant"
	webContext "github.com/golibs-starter/golib/web/context"
	webEvent "github.com/golibs-starter/golib/web/event"
	webLog "github.com/golibs-starter/golib/web/log"
	assert "github.com/stretchr/testify/require"
)

type TestProducer struct {
	message *core.Message
}

func (t *TestProducer) Send(m *core.Message) (partition int32, offset int64, err error) {
	t.message = m
	return 1, 0, nil
}

func (t *TestProducer) Close() error {
	return nil
}

type TestEvent struct {
	*webEvent.AbstractEvent
}

func newTestEvent(ctx context.Context, payload interface{}) *TestEvent {
	return &TestEvent{AbstractEvent: webEvent.NewAbstractEvent(ctx, "TestEvent", event.WithPayload(payload))}
}

type TestOrderableEvent struct {
	*webEvent.AbstractEvent
	PayloadData interface{} `json:"payload"`
	OrderId     string
}

func (t TestOrderableEvent) OrderingKey() string {
	return t.OrderId
}

func newTestOrderableEvent(ctx context.Context, payload interface{}) *TestOrderableEvent {
	return &TestOrderableEvent{
		AbstractEvent: webEvent.NewAbstractEvent(ctx, "TestOrderableEvent", event.WithPayload(payload)),
	}
}

func TestEventMessageRelayer_WhenTopicMappingNotExists_ShouldNotSupport(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	listener := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter)
	assert.False(t, listener.Supports(webEvent.NewAbstractEvent(context.Background(), "TestEvent")))
}

func TestEventMessageRelayer_WhenEventTopicIsDisabled_ShouldNotSupport(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": { // We provide lowercase here because golib properties always produce lowercase when binding map
			TopicName:     "test.topic",
			Transactional: false,
			Disable:       true,
		},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	listener := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter)
	assert.False(t, listener.Supports(webEvent.NewAbstractEvent(context.Background(), "TestEvent")))
}

func TestEventMessageRelayer_WhenEventTopicNameIsEmpty_ShouldNotSupport(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": { // We provide lowercase here because golib properties always produce lowercase when binding map
			TopicName: "",
		},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	listener := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter)
	assert.False(t, listener.Supports(webEvent.NewAbstractEvent(context.Background(), "TestEvent")))
}

func TestEventMessageRelayer_WhenEventTopicIsEnabled_ShouldSupport(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": { // We provide lowercase here because golib properties always produce lowercase when binding map
			TopicName: "test.topic",
		},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	listener := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter)
	assert.True(t, listener.Supports(webEvent.NewAbstractEvent(context.Background(), "TestEvent")))
}

func TestEventMessageRelayer_WhenIsApplicationEvent_ShouldSendMessageWithCorrectMessageAndHeaders(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testapplicationevent": {TopicName: "test.application.topic"},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	listener := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter)
	testEvent := event.NewApplicationEvent(context.Background(), "TestApplicationEvent")
	listener.Handle(testEvent)

	assert.NotNil(t, producer.message)
	assert.Equal(t, "test.application.topic", producer.message.Topic)

	expectedTestEventBytes, err := json.Marshal(testEvent)
	assert.NoError(t, err)
	assert.Equal(t, string(expectedTestEventBytes), string(producer.message.Value))
	assert.Nil(t, producer.message.Key)

	assert.Len(t, producer.message.Headers, 2)
	assert.Equal(t, constant.HeaderEventId, string(producer.message.Headers[0].Key))
	assert.Equal(t, testEvent.Identifier(), string(producer.message.Headers[0].Value))
	assert.Equal(t, constant.HeaderServiceClientName, string(producer.message.Headers[1].Key))
	assert.Equal(t, appProps.Name, string(producer.message.Headers[1].Value))
	assert.IsType(t, map[string]interface{}{}, producer.message.Metadata)
	assert.Len(t, producer.message.Metadata, 2)
	resultMetadata := producer.message.Metadata.(map[string]interface{})
	assert.Equal(t, testEvent.Identifier(), resultMetadata[kafkaConstant.EventId])
	assert.Equal(t, testEvent.Name(), resultMetadata[kafkaConstant.EventName])
	assert.Nil(t, resultMetadata[kafkaConstant.LoggingContext])
}

func TestEventMessageRelayer_WhenIsWebEvent_ShouldSendMessageWithCorrectMessageAndHeaders(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": {TopicName: "test.topic"},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	listener := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter)
	fakeRequestCtx := context.WithValue(context.Background(), constant.ContextReqAttribute, &webContext.RequestAttributes{
		CorrelationId:   "test-request-id",
		DeviceId:        "test-device-id",
		DeviceSessionId: "test-device-session-id",
		ClientIpAddress: "test-client-ip",
		SecurityAttributes: webContext.SecurityAttributes{
			UserId:            "test-user-id",
			TechnicalUsername: "test-technical-username",
		},
		ServiceCode: appProps.Name,
	})
	testEvent := newTestEvent(fakeRequestCtx, "TestEvent")
	listener.Handle(testEvent)

	assert.NotNil(t, producer.message)
	assert.Equal(t, "test.topic", producer.message.Topic)

	expectedTestEventBytes, err := json.Marshal(testEvent)
	assert.NoError(t, err)
	assert.Equal(t, string(expectedTestEventBytes), string(producer.message.Value))
	assert.Nil(t, producer.message.Key)

	assert.Len(t, producer.message.Headers, 6)
	assert.Equal(t, constant.HeaderEventId, string(producer.message.Headers[0].Key))
	assert.Equal(t, testEvent.Identifier(), string(producer.message.Headers[0].Value))
	assert.Equal(t, constant.HeaderServiceClientName, string(producer.message.Headers[1].Key))
	assert.Equal(t, appProps.Name, string(producer.message.Headers[1].Value))
	assert.Equal(t, constant.HeaderCorrelationId, string(producer.message.Headers[2].Key))
	assert.Equal(t, "test-request-id", string(producer.message.Headers[2].Value))
	assert.Equal(t, constant.HeaderDeviceId, string(producer.message.Headers[3].Key))
	assert.Equal(t, "test-device-id", string(producer.message.Headers[3].Value))
	assert.Equal(t, constant.HeaderDeviceSessionId, string(producer.message.Headers[4].Key))
	assert.Equal(t, "test-device-session-id", string(producer.message.Headers[4].Value))
	assert.Equal(t, constant.HeaderClientIpAddress, string(producer.message.Headers[5].Key))
	assert.Equal(t, "test-client-ip", string(producer.message.Headers[5].Value))
	assert.IsType(t, map[string]interface{}{}, producer.message.Metadata)
	assert.Len(t, producer.message.Metadata, 3)
	resultMetadata := producer.message.Metadata.(map[string]interface{})
	assert.Equal(t, testEvent.Identifier(), resultMetadata[kafkaConstant.EventId])
	assert.Equal(t, testEvent.Name(), resultMetadata[kafkaConstant.EventName])
	assert.IsType(t, &webLog.ContextAttributes{}, resultMetadata[kafkaConstant.LoggingContext])
	resultLoggingContext := resultMetadata[kafkaConstant.LoggingContext].(*webLog.ContextAttributes)
	assert.Equal(t, "test-user-id", resultLoggingContext.UserId)
	assert.Equal(t, "test-technical-username", resultLoggingContext.TechnicalUsername)
	assert.Equal(t, "test-device-id", resultLoggingContext.DeviceId)
	assert.Equal(t, "test-device-session-id", resultLoggingContext.DeviceSessionId)
	assert.Equal(t, "test-request-id", resultLoggingContext.CorrelationId)
}

func TestEventMessageRelayer_WhenEventIsOrderable_ShouldSendMessageWithCorrectKey(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testorderableevent": {TopicName: "test.topic"},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	listener := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter)
	testEvent := newTestOrderableEvent(context.Background(), "TestEvent")
	testEvent.OrderId = "3"
	listener.Handle(testEvent)

	assert.NotNil(t, producer.message)
	assert.Equal(t, "test.topic", producer.message.Topic)

	expectedTestEventBytes, err := json.Marshal(testEvent)
	assert.NoError(t, err)
	assert.Equal(t, string(expectedTestEventBytes), string(producer.message.Value))
	assert.Equal(t, "3", string(producer.message.Key))
}

func TestEventMessageRelayer_WhenIsWebEventAndNotLogPayload_ShouldSuccess(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": {TopicName: "test.topic"},
	}}
	eventProps := &event.Properties{
		Log: event.LogProperties{
			NotLogPayloadForEvents: []string{"TestEvent"},
		},
	}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	listener := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter)
	testEvent := webEvent.NewAbstractEvent(context.Background(), "TestEvent")
	listener.Handle(testEvent)
	assert.NotNil(t, producer.message)
}

func TestEventMessageRelayer_RegisterHandler(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": {TopicName: "test.topic"},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	relayer := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter).(*EventMessageRelayer)

	// Test registering a single handler
	handlerCalled := false
	relayer.RegisterHandler("test.topic", func(msg *core.ConsumerMessage) error {
		handlerCalled = true
		assert.Equal(t, "test-message", string(msg.Value))
		return nil
	})

	// Verify handler was registered
	assert.Len(t, relayer.handlers["test.topic"], 1)

	// Test handler execution
	err := relayer.HandleMessage(&core.ConsumerMessage{
		Topic: "test.topic",
		Value: []byte("test-message"),
	})
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestEventMessageRelayer_MultipleHandlers(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": {TopicName: "test.topic"},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	relayer := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter).(*EventMessageRelayer)

	// Track handler calls
	handler1Called := false
	handler2Called := false

	// Register multiple handlers
	relayer.RegisterHandler("test.topic", func(msg *core.ConsumerMessage) error {
		handler1Called = true
		return nil
	})

	relayer.RegisterHandler("test.topic", func(msg *core.ConsumerMessage) error {
		handler2Called = true
		return nil
	})

	// Verify both handlers were registered
	assert.Len(t, relayer.handlers["test.topic"], 2)

	// Test handlers execution
	err := relayer.HandleMessage(&core.ConsumerMessage{
		Topic: "test.topic",
		Value: []byte("test-message"),
	})
	assert.NoError(t, err)
	assert.True(t, handler1Called)
	assert.True(t, handler2Called)
}

func TestEventMessageRelayer_HandlerError(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": {TopicName: "test.topic"},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	relayer := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter).(*EventMessageRelayer)

	// Register handler that returns error
	expectedError := errors.New("handler error")
	relayer.RegisterHandler("test.topic", func(msg *core.ConsumerMessage) error {
		return expectedError
	})

	// Test handler execution
	err := relayer.HandleMessage(&core.ConsumerMessage{
		Topic: "test.topic",
		Value: []byte("test-message"),
	})
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestEventMessageRelayer_NoHandlersForTopic(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": {TopicName: "test.topic"},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	relayer := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter).(*EventMessageRelayer)

	// Test handling message for topic with no handlers
	err := relayer.HandleMessage(&core.ConsumerMessage{
		Topic: "nonexistent.topic",
		Value: []byte("test-message"),
	})
	assert.NoError(t, err)
}

// Add this test
func TestEventMessageRelayer_RegisterTypedHandler(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": {TopicName: "test.topic"},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	relayer := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter).(*EventMessageRelayer)

	// Test struct
	type UserEvent struct {
		UserID   string            `json:"userId"`
		Action   string            `json:"action"`
		Metadata map[string]string `json:"metadata"`
	}

	// Test registering a typed handler
	var receivedEvent UserEvent
	RegisterTypedHandler(relayer, "test.topic", func(event UserEvent) error {
		receivedEvent = event
		return nil
	})

	// Create test message
	testEvent := UserEvent{
		UserID: "123",
		Action: "login",
		Metadata: map[string]string{
			"ip": "127.0.0.1",
		},
	}
	testEventBytes, err := json.Marshal(testEvent)
	assert.NoError(t, err)

	// Test handler execution
	err = relayer.HandleMessage(&core.ConsumerMessage{
		Topic: "test.topic",
		Value: testEventBytes,
	})
	assert.NoError(t, err)

	// Verify parsed event
	assert.Equal(t, testEvent.UserID, receivedEvent.UserID)
	assert.Equal(t, testEvent.Action, receivedEvent.Action)
	assert.Equal(t, testEvent.Metadata["ip"], receivedEvent.Metadata["ip"])
}

// Add test for unmarshal error
func TestEventMessageRelayer_RegisterTypedHandler_UnmarshalError(t *testing.T) {
	producer := &TestProducer{}
	appProps := &config.AppProperties{Name: "TestApp"}
	eventProducerProps := &properties.EventProducer{EventMappings: map[string]properties.EventTopic{
		"testevent": {TopicName: "test.topic"},
	}}
	eventProps := &event.Properties{}
	converter := NewDefaultEventConverter(appProps, eventProducerProps)
	relayer := NewEventMessageRelayer(producer, eventProducerProps, eventProps, converter).(*EventMessageRelayer)

	type UserEvent struct {
		UserID string `json:"userId"`
	}

	// Register typed handler
	RegisterTypedHandler(relayer, "test.topic", func(event UserEvent) error {
		return nil
	})

	// Test with invalid JSON
	err := relayer.HandleMessage(&core.ConsumerMessage{
		Topic: "test.topic",
		Value: []byte("invalid json"),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal message")
}
