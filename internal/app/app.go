// Package app wires together all components and runs the HTTP server
// with a RabbitMQ consumer for delayed notifications.
package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/venexene/gong/internal/config"
	"github.com/venexene/gong/internal/handler"
	"github.com/venexene/gong/internal/queue"
	"github.com/venexene/gong/internal/repository"
)

// Run starts the application
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("failed to load config: %v", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := repository.New(ctx, cfg.DB_DSN)
	if err != nil {
		log.Printf("failed to create repository: %v", err)
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer db.Pool.Close()

	rabbit, err := queue.New(cfg.RabbitURL)
	if err != nil {
		log.Printf("failed to connect to RabbitMQ: %v", err)
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.GET("/test_server", handler.HealthCheck)
	router.POST("/notify", handler.CreateNotification(db, rabbit))
	router.GET("/notify/:id", handler.GetNotificationStatus(db))
	router.DELETE("/notify/:id", handler.CancelNotification(db))

	srv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: router,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("server starting on http://localhost%s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("failed to start server: %v", err)
			errCh <- fmt.Errorf("failed to start server: %w", err)
		}
	}()

	notifier := queue.LogNotifier{}
	go rabbit.Consume(ctx, db, notifier)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Println("Shutting down server...")

		ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctxShutdown); err != nil {
			log.Printf("failed to shutdown server: %v", err)
			return fmt.Errorf("failed to shutdown server: %w", err)
		}

		log.Println("Server shutdown completed")
	}

	return nil
}
