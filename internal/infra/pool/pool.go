package pool

import (
	"context"
	"encoding/json"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"notifications-service/internal/core/dispatcher"
	"notifications-service/internal/core/domain"
	"notifications-service/internal/infra/queue"
)

type WorkerPool struct {
	queue      queue.Queue
	dispatcher dispatcher.EventDispatcher
	numWorkers int
}

func NewWorkerPool(
	queue queue.Queue,
	dispatcher dispatcher.EventDispatcher,
	numWorkers int,
) *WorkerPool {
	return &WorkerPool{
		queue:      queue,
		dispatcher: dispatcher,
		numWorkers: numWorkers,
	}
}

func (pool *WorkerPool) Start(ctx context.Context) {
	messages, err := pool.queue.Consume(ctx)
	if err != nil {
		slog.Error("Error consuming messages from queue", "error", err)
		return
	}

	slog.Info("Starting worker")
	g, gCtx := errgroup.WithContext(ctx)
	for i := 0; i < pool.numWorkers; i++ {
		g.Go(func() error {
			return pool.worker(gCtx, messages)
		})
	}

	if waitErr := g.Wait(); waitErr != nil {
		slog.Error("Error consuming messages from queue", "error", waitErr)
	}
}

func (pool *WorkerPool) worker(ctx context.Context, messages <-chan queue.Message) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message, ok := <-messages:
			if !ok {
				slog.Info("Messages channel closed, worker exiting")
				return nil
			}
			pool.handleMessage(ctx, message)
		}
	}
}

func (pool *WorkerPool) handleMessage(ctx context.Context, message queue.Message) {
	var notification domain.NotificationEnvelope

	if err := json.Unmarshal(message.Body(), &notification); err != nil {
		slog.Error("Error unmarshalling message", "error", err)
		pool.nackMessage(message)
		return
	}

	if err := pool.dispatcher.Dispatch(ctx, notification); err != nil {
		slog.Error("Error dispatching message", "error", err)
		pool.nackMessage(message)
		return
	}

	if err := message.Ack(); err != nil {
		slog.Error("Failed to ack message", "error", err)
	}
}

func (pool *WorkerPool) nackMessage(message queue.Message) {
	if err := message.Nack(false, false); err != nil {
		slog.Error("Failed to nack message", "error", err)
	}
}
