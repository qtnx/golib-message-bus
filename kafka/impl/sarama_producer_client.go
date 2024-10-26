package impl

import (
	"github.com/IBM/sarama"
	"github.com/golibs-starter/golib-message-bus/kafka/properties"
	"github.com/pkg/errors"
)

func NewSaramaProducerClient(
	globalProps *properties.Client,
	existingConfig *sarama.Config,
) (sarama.Client, error) {
	config, err := CreateCommonSaramaConfig(existingConfig, globalProps.Version, globalProps.Producer)
	if err != nil {
		return nil, errors.WithMessage(err, "Create sarama config error")
	}
	props := globalProps.Producer
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.Flush.Messages = props.FlushMessages
	config.Producer.Flush.Frequency = props.FlushFrequency
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	if err := config.Validate(); err != nil {
		return nil, errors.WithMessage(err, "Error when validate producer client config")
	}
	client, err := sarama.NewClient(props.BootstrapServers, config)
	if err != nil {
		return nil, errors.WithMessage(err, "Error when create sarama producer client")
	}
	return client, nil
}
