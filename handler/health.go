package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthChecker interface {
	Ping(ctx context.Context) error
}

func Health(checker HealthChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := checker.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	}
}
