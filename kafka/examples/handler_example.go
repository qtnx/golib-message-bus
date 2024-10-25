package examples

import (
	"fmt"

	"github.com/golibs-starter/golib-message-bus/kafka/relayer"
	"go.uber.org/fx"
)

type HandlerDeps struct {
	fx.In
	Relayer *relayer.EventMessageRelayer
}

type UserEvent struct {
	UserID   string            `json:"userId"`
	Action   string            `json:"action"`
	Metadata map[string]string `json:"metadata"`
}

type OrderEvent struct {
	OrderID     string  `json:"orderId"`
	TotalAmount float64 `json:"totalAmount"`
	Status      string  `json:"status"`
}

func RegisterHandlers(deps HandlerDeps) {
	// Register typed handlers
	relayer.RegisterTypedHandler(deps.Relayer, "user-events", handleTypedUserEvent)
	relayer.RegisterTypedHandler(deps.Relayer, "order-events", handleTypedOrderEvent)
}

func handleTypedUserEvent(event UserEvent) error {
	fmt.Printf("Processing user event: UserID=%s, Action=%s\n",
		event.UserID, event.Action)
	return nil
}

func handleTypedOrderEvent(event OrderEvent) error {
	fmt.Printf("Processing order event: OrderID=%s, Amount=%.2f, Status=%s\n",
		event.OrderID, event.TotalAmount, event.Status)
	return nil
}

// Usage with fx
func KafkaHandlersOpt() fx.Option {
	return fx.Options(
		fx.Invoke(RegisterHandlers),
	)
}
