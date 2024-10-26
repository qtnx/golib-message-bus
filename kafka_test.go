package golibmsg

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/golibs-starter/golib"
	"github.com/golibs-starter/golib-message-bus/kafka/impl"
	"github.com/golibs-starter/golib-message-bus/kafka/properties"
	"github.com/golibs-starter/golib/config"
	"github.com/golibs-starter/golib/event"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func NewPropertiesLoader() (config.Loader, error) {
	loader, err := config.NewLoader(config.Option{
		ConfigPaths: []string{"testdata/kafka.yml"},
	}, []config.Properties{
		&properties.Client{},
	})
	return loader, err
}

func TestKafkaProducerWithConfigOpt(t *testing.T) {
	t.Run("should apply custom config when configFn is provided", func(t *testing.T) {
		// Given
		expectedVersion := sarama.V3_0_0_0
		configFn := func(config *sarama.Config) {
			config.Version = expectedVersion
		}

		// When
		app := fxtest.New(t,
			fx.Provide(NewPropertiesLoader),
			fx.Provide(func() *event.Properties {
				return &event.Properties{}
			}),
			golib.ProvideProps(func() *properties.Client {
				return &properties.Client{
					BootstrapServers: []string{"localhost:9092"},
					Version:          "3.0.0",
				}
			}),
			// Add SaramaMapper dependency
			fx.Provide(impl.NewSaramaMapper),
			// Add DebugLogger dependency
			fx.Provide(impl.NewDebugLogger),
			// KafkaProducerOpt(),
			KafkaProducerWithConfigOpt(configFn),
			fx.Invoke(func(config *sarama.Config) {
				// Then
				assert.Equal(t, expectedVersion, config.Version)
			}),
		)
		app.RequireStart()
		defer app.RequireStop()
	})

	t.Run("should use default config when configFn is nil", func(t *testing.T) {
		// When
		app := fxtest.New(t,
			fx.Provide(NewPropertiesLoader),
			// Provide required dependencies
			golib.ProvideProps(func() *properties.Client {
				return &properties.Client{
					BootstrapServers: []string{"localhost:9092"},
					Version:          "2.0.0",
				}
			}),
			fx.Provide(func() *event.Properties {
				return &event.Properties{}
			}),
			// Add SaramaMapper dependency
			fx.Provide(impl.NewSaramaMapper),
			// Add DebugLogger dependency
			fx.Provide(impl.NewDebugLogger),
			KafkaProducerOpt(),
			// KafkaProducerWithConfigOpt(nil),
			fx.Invoke(func(config *sarama.Config) {
				// Then
				assert.NotNil(t, config)
				assert.Equal(t, sarama.V2_0_0_0, config.Version)
			}),
		)
		app.RequireStart()
		defer app.RequireStop()
	})
}

func TestKafkaConsumerWithConfigOpt(t *testing.T) {
	t.Run("should apply custom config when configFn is provided", func(t *testing.T) {
		// Given
		expectedVersion := sarama.V3_0_0_0
		configFn := func(config *sarama.Config) {
			config.Version = expectedVersion
		}

		// When
		app := fxtest.New(t,
			// Provide required dependencies
			fx.Provide(func() *properties.Client {
				return &properties.Client{
					BootstrapServers: []string{"localhost:9092"},
					Version:          "3.0.0",
				}
			}),
			// Add SaramaMapper dependency
			fx.Provide(impl.NewSaramaMapper),
			KafkaConsumerWithConfigOpt(configFn),
			fx.Invoke(func(config *sarama.Config) {
				// Then
				assert.Equal(t, expectedVersion, config.Version)
			}),
		)
		app.RequireStart()
		defer app.RequireStop()
	})

	t.Run("should use default config when configFn is nil", func(t *testing.T) {
		// When
		app := fxtest.New(t,
			// Provide required dependencies
			fx.Provide(func() *properties.Client {
				return &properties.Client{
					BootstrapServers: []string{"localhost:9092"},
					Version:          "2.0.0",
				}
			}),
			// Add SaramaMapper dependency
			fx.Provide(impl.NewSaramaMapper),
			KafkaConsumerWithConfigOpt(nil),
			fx.Invoke(func(config *sarama.Config) {
				// Then
				assert.NotNil(t, config)
				assert.Equal(t, sarama.V2_0_0_0, config.Version)
			}),
		)
		app.RequireStart()
		defer app.RequireStop()
	})
}
