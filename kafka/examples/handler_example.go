package examples

import (
	"fmt"
	"github.com/golibs-starter/golib-message-bus/kafka/core"
	"github.com/golibs-starter/golib-message-bus/kafka/relayer"
	"go.uber.org/fx"
)

type HandlerDeps struct {
	fx.In
	Relayer *relayer.EventMessageRelayer
}

func RegisterHandlers(deps HandlerDeps) {
	// Register handlers for specific topics
	deps.Relayer.RegisterHandler("user-events", handleUserEvent)
	deps.Relayer.RegisterHandler("user-events", auditLogUserEvent)
}

func handleUserEvent(msg *core.ConsumerMessage) error {
	fmt.Printf("Processing user event: %s\n", string(msg.Value))
	return nil
}

func auditLogUserEvent(msg *core.ConsumerMessage) error {
	fmt.Printf("Audit logging: %s\n", string(msg.Value))
	return nil
}

// Usage with fx
func KafkaHandlersOpt() fx.Option {
	return fx.Options(
		fx.Invoke(RegisterHandlers),
	)
}
