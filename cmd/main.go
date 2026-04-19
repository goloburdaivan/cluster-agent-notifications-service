package main

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"log/slog"
	"net/http"
	"notifications-service/internal/api/handlers"
	"notifications-service/internal/api/middleware"
	"notifications-service/internal/infra/crypto"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"notifications-service/internal/core/dispatcher"
	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/listeners"
	"notifications-service/internal/core/services"

	"notifications-service/internal/infra/clients/email"
	"notifications-service/internal/infra/clients/slack"
	"notifications-service/internal/infra/clients/telegram"
	"notifications-service/internal/infra/pool"
	"notifications-service/internal/infra/providers"
	"notifications-service/internal/infra/queue"
	"notifications-service/internal/infra/repository"
	infraTemplate "notifications-service/internal/infra/template"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	if err := db.Ping(ctx); err != nil {
		log.Fatal("Failed to ping DB:", err)
	}
	defer db.Close()

	key, err := hex.DecodeString(os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		log.Fatalf("Failed to decode encryption key: %v", err)
	}

	encrypter, err := crypto.NewAESEncrypter(key)
	if err != nil {
		log.Fatal("Failed to create encrypter:", err)
	}

	channelRepository := repository.NewChannelRepository(db, encrypter)
	channelService := services.NewChannelService(channelRepository)
	handler := handlers.NewChannelHandler(channelService)

	router := gin.Default()
	router.Use(middleware.ErrorHandler())
	v1 := router.Group("/v1")
	channels := v1.Group("/channels")
	{
		channels.GET("", handler.List)
		channels.GET("/:id", handler.Get)
		channels.POST("", handler.Create)
		channels.PATCH("/:id", handler.Update)
		channels.DELETE("/:id", handler.Delete)
	}

	server := http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	renderer, err := infraTemplate.NewRenderer("templates")
	if err != nil {
		log.Fatal("Failed to init template renderer:", err)
	}

	customHTTPClient := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			MaxIdleConns:        100,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     60 * time.Second,
		},
	}
	tgClient := telegram.NewClient(telegram.WithHTTPClient(customHTTPClient))
	telegramSender := providers.NewSender(tgClient, renderer)

	slackClient := slack.NewClient(slack.WithHTTPClient(customHTTPClient))
	slackSender := providers.NewSlackSender(slackClient, renderer)

	emailClient := email.NewClient()
	emailSender := providers.NewEmailSender(emailClient, renderer)

	notificationRouter := services.NewNotificationRouter(telegramSender, slackSender, emailSender)
	securityHandler := listeners.NewSecurityHandler(channelRepository, notificationRouter)

	conn, err := amqp.Dial(os.Getenv("RABBIT_URL"))
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer conn.Close()

	notificationQueue := queue.NewRabbitQueue("notifications", conn)
	defer func() {
		if err := notificationQueue.Close(); err != nil {
			slog.Error("Failed to close notification queue", "error", err)
		}
	}()

	eventDispatcher := dispatcher.NewEventDispatcher()
	eventDispatcher.Register(domain.EventTypeSecurity, securityHandler.Handle)

	worker := pool.NewWorkerPool(notificationQueue, eventDispatcher, 10)
	slog.Info("Starting notification worker")

	go worker.Start(ctx)

	<-ctx.Done()
	logger.Info("Shutting down gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Failed to shutdown gracefully:", err)
	}
}
