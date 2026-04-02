package queue

import (
	"context"
	"io"
)

type Queue interface {
	io.Closer
	Consume(ctx context.Context) (<-chan Message, error)
}
