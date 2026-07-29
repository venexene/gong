package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/venexene/wbl3-delayed-notifier/internal/config"
	"github.com/venexene/wbl3-delayed-notifier/internal/handler"
	"github.com/venexene/wbl3-delayed-notifier/internal/queue"
	"github.com/venexene/wbl3-delayed-notifier/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()

	db, err := storage.New(ctx, cfg.DB_DSN)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}
	defer db.Pool.Close()

	rabbit, err := queue.New(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
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

	go func() {
		log.Printf("server starting on http://localhost%s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	ctxStop, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	notifier := queue.LogNotifier{}
	go rabbit.Consume(ctxStop, db, notifier)

	<-ctxStop.Done()
	log.Println("Shutting down server...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("failed to shutdown server: %v", err)
	}

	log.Println("Server shutdown completed")
}
