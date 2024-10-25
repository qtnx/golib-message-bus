package impl

import (
	"github.com/IBM/sarama"
	"github.com/golibs-starter/golib-message-bus/kafka/core"
	"github.com/golibs-starter/golib/log"
	"github.com/pkg/errors"
)

type SaramaAsyncProducer struct {
	producer    sarama.AsyncProducer
	errorsCh    chan *core.ProducerError
	successesCh chan *core.Message
	mapper      *SaramaMapper
}

func NewSaramaAsyncProducer(client sarama.Client, mapper *SaramaMapper) (*SaramaAsyncProducer, error) {
	asyncProducer, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		return nil, errors.WithMessage(err, "Error when create new async producer")
	}
	p := &SaramaAsyncProducer{
		producer:    asyncProducer,
		errorsCh:    make(chan *core.ProducerError),
		successesCh: make(chan *core.Message),
		mapper:      mapper,
	}
	go func() {
		for e := range asyncProducer.Successes() {
			p.successesCh <- p.mapper.ToCoreMessage(e)
		}
	}()
	go func() {
		for e := range asyncProducer.Errors() {
			p.errorsCh <- &core.ProducerError{
				Msg: p.mapper.ToCoreMessage(e.Msg),
				Err: e.Err,
			}
		}
	}()
	return p, nil
}

func (p *SaramaAsyncProducer) Send(m *core.Message) {
	msg := &sarama.ProducerMessage{
		Topic:    m.Topic,
		Value:    sarama.ByteEncoder(m.Value),
		Headers:  p.mapper.ToSaramaHeaders(m.Headers),
		Metadata: m.Metadata,
	}
	if m.Key != nil {
		msg.Key = sarama.ByteEncoder(m.Key)
	}
	p.producer.Input() <- msg
}

func (p *SaramaAsyncProducer) Successes() <-chan *core.Message {
	return p.successesCh
}

func (p *SaramaAsyncProducer) Errors() <-chan *core.ProducerError {
	return p.errorsCh
}

func (p *SaramaAsyncProducer) Close() error {
	log.Info("Kafka async producer is stopping")
	defer log.Info("Kafka async producer is stopped")
	return p.producer.Close()
}
