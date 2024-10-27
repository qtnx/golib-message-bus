package global

import (
	"strings"
	"sync"
)

type IEventTopicMapping interface {
	GetTopic(event string) string
	SetTopic(event, topic string)
	UnsetTopic(event string)
}

type eventTopicMapping struct {
	// key: event name, value: topic name
	topics map[string]string
	lock   sync.RWMutex
}

func NewEventTopicMapping() IEventTopicMapping {
	return &eventTopicMapping{
		topics: make(map[string]string),
	}
}

func (e *eventTopicMapping) GetTopic(event string) string {
	e.lock.RLock()
	defer e.lock.RUnlock()
	eventName := strings.ToLower(event)
	return e.topics[eventName]
}

func (e *eventTopicMapping) SetTopic(event, topic string) {
	e.lock.Lock()
	defer e.lock.Unlock()
	eventName := strings.ToLower(event)
	e.topics[eventName] = topic
}

func (e *eventTopicMapping) UnsetTopic(event string) {
	e.lock.Lock()
	defer e.lock.Unlock()
	eventName := strings.ToLower(event)
	delete(e.topics, eventName)
}

var EventTopicMappingInstance IEventTopicMapping

func init() {
	once := sync.Once{}
	once.Do(func() {
		EventTopicMappingInstance = NewEventTopicMapping()
	})
}
