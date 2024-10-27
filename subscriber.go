package golibmsg

import (
	"context"
	"encoding/json"

	"github.com/golibs-starter/golib-message-bus/kafka/core"
	"github.com/golibs-starter/golib-message-bus/kafka/global"
	"github.com/golibs-starter/golib/log"
	"github.com/golibs-starter/golib/pubsub"
)

// Subscribe to topic with event callback
func SubscribeToTopic[T any](ctx context.Context, topic string, handler func(msg T)) {
	global.SubscriberTopicInstance.Register(topic, func(msg *core.ConsumerMessage) {
		var event pubsub.MessageEvent[T]
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Errorf("Error when unmarshal message: %v", err)
			return
		}
		handler(event.PayloadData)
	})
}

// Subscribe to topic with raw message callback
func SubscribeToTopicRawMsg(ctx context.Context, topic string, handler func(msg *core.ConsumerMessage)) {
	global.SubscriberTopicInstance.Register(topic, handler)
}

// Subscribe to topic with raw message callback and close handler
func SubscribeToTopicRawMsgWithClose(ctx context.Context, topic string, handler func(msg *core.ConsumerMessage), closeHandler func()) {
	global.SubscriberTopicInstance.RegisterWithClose(topic, handler, closeHandler)
}
