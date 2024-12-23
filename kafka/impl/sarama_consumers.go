package impl

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/golibs-starter/golib-message-bus/kafka/core"
	"github.com/golibs-starter/golib-message-bus/kafka/global"
	"github.com/golibs-starter/golib-message-bus/kafka/properties"
	"github.com/golibs-starter/golib/log"
	"github.com/pkg/errors"
)

type SaramaConsumers struct {
	clientProps        *properties.Client
	consumerProps      *properties.Consumer
	kafkaConsumerProps *properties.KafkaConsumer
	mapper             *SaramaMapper
	consumers          map[string]*SaramaConsumer
	unready            chan bool
}

func NewSaramaConsumers(
	clientProps *properties.Client,
	consumerProps *properties.KafkaConsumer,
	mapper *SaramaMapper,
	handlers []core.ConsumerHandler,
	existingConfig *sarama.Config,
) (*SaramaConsumers, error) {
	if len(consumerProps.HandlerMappings) < 1 {
		log.Warn("[SaramaConsumers] Missing handler mapping")
		// return nil, nil
		consumerProps.HandlerMappings = map[string]properties.TopicConsumer{}
	}
	log.Infof("[SaramaConsumers] Handler mappings: %v", consumerProps.HandlerMappings)

	handlerMap := make(map[string]core.ConsumerHandler)
	for _, handler := range handlers {
		handlerMap[strings.ToLower(getStructShortName(handler))] = handler
	}

	kafkaConsumers := SaramaConsumers{
		clientProps:        clientProps,
		consumerProps:      &clientProps.Consumer,
		kafkaConsumerProps: consumerProps,
		mapper:             mapper,
		consumers:          make(map[string]*SaramaConsumer),
		unready:            make(chan bool),
	}

	if err := kafkaConsumers.init(handlerMap, existingConfig); err != nil {
		return nil, errors.WithMessage(err, "[SaramaConsumers] Error when init kafka consumers")
	}

	return &kafkaConsumers, nil
}

func (s *SaramaConsumers) init(
	handlerMap map[string]core.ConsumerHandler,
	existingConfig *sarama.Config,
) error {
	for key, config := range s.kafkaConsumerProps.HandlerMappings {
		if !config.Enable {
			log.Debugf("Kafka consumer key [%s] is not enabled", key)
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		handler, exists := handlerMap[key]
		if !exists {
			if h := global.SubscriberTopicInstance.GetHandler(key); h != nil {
				handler = h
			} else {
				log.Debugf("Kafka consumer key [%s] is not exists in handler list", key)
				continue
			}
		}
		saramaConsumer, err := NewSaramaConsumer(
			s.mapper,
			s.clientProps,
			&config,
			handler,
			existingConfig,
		)
		if err != nil {
			return err
		}
		s.consumers[key] = saramaConsumer
	}
	return nil
}

func (s *SaramaConsumers) Start(ctx context.Context) {
	wg := &sync.WaitGroup{}
	wg.Add(len(s.consumers))
	for _, consumer := range s.consumers {
		go func(consumer *SaramaConsumer) {
			defer wg.Done()
			go consumer.Start(ctx)
			<-consumer.WaitForReady()
		}(consumer)
	}
	wg.Wait()
	close(s.unready)
}

func (s SaramaConsumers) WaitForReady() chan bool {
	return s.unready
}

func (s *SaramaConsumers) Stop() {
	var wg sync.WaitGroup
	wg.Add(len(s.consumers))
	for _, consumer := range s.consumers {
		go func(consumer *SaramaConsumer) {
			defer wg.Done()
			consumer.Stop()
		}(consumer)
	}
	wg.Wait()
}

func getStructShortName(val interface{}) string {
	if val == nil {
		return ""
	}
	if t := reflect.TypeOf(val); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else if t.Kind() == reflect.Struct {
		// Check if the Name method exists
		if nameMethod, ok := t.MethodByName("Name"); ok {
			v := reflect.ValueOf(val)
			// Call the Name method and return its result if it returns a single string
			results := nameMethod.Func.Call([]reflect.Value{v})
			if len(results) == 1 && results[0].Kind() == reflect.String {
				return results[0].String()
			}
		}
		return t.Name()
	} else {
		return ""
	}
}
