// Package handler provides HTTP handlers for the delayed notification API.
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/venexene/gong/internal/queue"
	"github.com/venexene/gong/internal/repository"
)

// HealthCheck returns a simple health-check response.
func HealthCheck(c *gin.Context) {
	c.String(http.StatusOK, "Hello! Server is running. Time: %s", time.Now().Format(time.RFC1123))
}

// CreateNotification returns a Gin handler that creates a delayed notification.
func CreateNotification(db repository.Store, q queue.Publisher) gin.HandlerFunc {
	type CreateRequest struct {
		Target  string    `json:"target"`
		Message string    `json:"message"`
		SendAt  time.Time `json:"send_at"`
	}

	return func(c *gin.Context) {
		var req CreateRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		n := repository.Notification{
			ID:      uuid.New().String(),
			Target:  req.Target,
			Message: req.Message,
			SendAt:  req.SendAt,
			Status:  "pending",
		}

		if err := db.Create(c, n); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := q.Publish(n); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": n.ID})
	}
}

// GetNotificationStatus returns a Gin handler that retrieves a notification by its ID.
func GetNotificationStatus(db repository.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		n, err := db.GetByID(c, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.JSON(http.StatusOK, n)
	}
}

// CancelNotification returns a Gin handler that cancels a pending notification.
func CancelNotification(db repository.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		ok, err := db.Cancel(c, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot cancel"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "canceled"})
	}
}
