package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/golibs-starter/golib"
	golibmsg "github.com/golibs-starter/golib-message-bus"
	"github.com/golibs-starter/golib-message-bus/kafka/core"
	"github.com/golibs-starter/golib-message-bus/kafka/properties"
	"github.com/golibs-starter/golib/config"
	"github.com/golibs-starter/golib/log"
	"github.com/golibs-starter/golib/pubsub"
	"go.uber.org/fx"
)

type HandlerDeps struct {
	fx.In
	Relayer       pubsub.Subscriber
	SyncProducer  core.SyncProducer
	AsyncProducer core.AsyncProducer
}

type UserEventData struct {
	UserID   string            `json:"userId"`
	Action   string            `json:"action"`
	Metadata map[string]string `json:"metadata"`
}

type OrderEvent struct {
	OrderID     string  `json:"orderId"`
	TotalAmount float64 `json:"totalAmount"`
	Status      string  `json:"status"`
}

func handleTypedUserEvent(event UserEventData) {
	fmt.Printf("[MSG] Processing user event: UserID=%s, Action=%s\n",
		event.UserID, event.Action)
}

func handleTypedOrderEvent(event OrderEvent) {
	fmt.Printf("[MSG] Processing order event: OrderID=%s, Amount=%.2f, Status=%s\n",
		event.OrderID, event.TotalAmount, event.Status)
}

func PublishEvents() {
	// Publish some example events

	orderEvent := OrderEvent{
		OrderID:     "order456",
		TotalAmount: 99.99,
		Status:      "completed",
	}
	fmt.Printf("Publishing order event: %+v\n", orderEvent)

	// Publish using sync producer
	// pubsub.Publish(msg)

	// Publish using async producer
	golibmsg.PublishToTopic(context.Background(), "orders", orderEvent)
	golibmsg.PublishToTopic(context.Background(), "user-events", UserEventData{
		UserID: "user123",
		Action: "login",
		Metadata: map[string]string{
			"ip": "192.168.1.1",
		},
	})

	fmt.Println("Published events")
}

type ManualConfig struct {
	KafkaVersion string `default:"2.1.1"`
}

func NewManualConfig() config.Loader {
	props := ManualConfig{}
	return &props
}

func (c *ManualConfig) Bind(props ...config.Properties) error {
	return nil
}

func (c *ManualConfig) PostBinding() error {
	return nil
}

// Application provides all the dependencies our application needs
func Application() fx.Option {
	return fx.Options(
		fx.Provide(NewManualConfig),

		fx.Provide(func() *config.AppProperties {
			return &config.AppProperties{}
		}),

		// Kafka options
		fx.Provide(func() *properties.Client {
			config := &properties.Client{}
			fmt.Printf("config before: %+v\n", config)
			config.Version = "2.1.1"
			config.BootstrapServers = []string{"localhost:29092"}
			config.Producer.ClientId = "kafka-handler-example"
			config.Producer.BootstrapServers = []string{"localhost:29092"}
			config.Consumer.BootstrapServers = []string{"localhost:29092"}
			config.Consumer.CommitMode = "AUTO_COMMIT_IMMEDIATELY"
			config.Consumer.InitialOffset = sarama.OffsetOldest
			config.Consumer.ClientId = "kafka-handler-example-group"
			config.Debug = true
			fmt.Printf("config after: %+v\n", config)
			return config
		}),
		fx.Provide(func() *sarama.Config {
			config := sarama.NewConfig()
			config.ClientID = "kafka-handler-example"
			return config
		}),
		fx.Provide(func() context.Context {
			return context.Background()
		}),
		golibmsg.KafkaCommonOptNoClient(),
		golibmsg.KafkaProducerOpt(),
		golibmsg.KafkaConsumerOpt(),
		golibmsg.OnStopProducerOpt(),
		golib.EventOpt(),
		golib.OnStopEventOpt(),
		golibmsg.OnStopConsumerOpt(),

		// Register handlers
		// fx.Invoke(RegisterHandlers),

		// Start publishing events
		fx.Invoke(func(lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					fmt.Println("Starting application")
					// Wait a bit for everything to initialize
					go func() {
						fmt.Println("Waiting 2 seconds before publishing events")
						time.Sleep(2 * time.Second)
						PublishEvents()
					}()
					return nil
				},
			})
		}),
	)
}

func init() {
	golibmsg.SubscribeToTopic(context.Background(), "orders", handleTypedOrderEvent)

	golibmsg.SubscribeToTopicRawMsg(context.Background(), "user-events", func(msg *core.ConsumerMessage) {
		fmt.Printf("Received message: %+v\n", msg)
		var event UserEventData
		err := json.Unmarshal(msg.Value, &event)
		if err != nil {
			log.WithErrors(err).Errorf("Error when unmarshal user event")
		}
		handleTypedUserEvent(event)
	})
}

func main() {
	app := fx.New(Application())
	app.Run()
}
