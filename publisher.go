package golibmsg

import (
	"context"
	"reflect"

	"github.com/golibs-starter/golib-message-bus/kafka/global"
	"github.com/golibs-starter/golib/pubsub"
)

func PublishEvent[T any](ctx context.Context, eventData T) {
	pubsub.PublishEvent(ctx, eventData)
}

func PublishToTopic[T any](ctx context.Context, topic string, eventData T) {
	eventName := reflect.TypeOf(eventData).Name()
	global.EventTopicMappingInstance.SetTopic(eventName, topic)
	pubsub.PublishEvent(ctx, eventData)
}
