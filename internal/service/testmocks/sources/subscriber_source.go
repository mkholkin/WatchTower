package sources

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Subscriber mirrors message.Subscriber for stable mock generation.
type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error)
	Close() error
}
