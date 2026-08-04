// Gong is an HTTP service that accepts delayed notifications and delivers them
// at the specified time via RabbitMQ TTL + dead-letter exchange.
package main

import (
	"log"

	"github.com/venexene/gong/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
	}
}
