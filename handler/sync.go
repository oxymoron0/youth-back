package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"kr-metro-api/repository"
	"kr-metro-api/sync"
)

type SyncHandler struct {
	housingSync *sync.HousingSync
	repo        repository.HousingRepository
	adminKey    string
}

func NewSyncHandler(housingSync *sync.HousingSync, repo repository.HousingRepository, adminKey string) *SyncHandler {
	return &SyncHandler{housingSync: housingSync, repo: repo, adminKey: adminKey}
}

func (h *SyncHandler) TriggerHousingSync(c *gin.Context) {
	if !h.authorize(c) {
		return
	}

	result := h.housingSync.RunOnce(c.Request.Context())
	if result.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"result": result,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"result": result,
	})
}

func (h *SyncHandler) SyncStatus(c *gin.Context) {
	if !h.authorize(c) {
		return
	}

	last := h.housingSync.LastResult()
	if last == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "no sync has been executed yet",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"result": last,
	})
}

// PublicLatestStatus returns the most recent sync result without auth.
// Falls back to DB when in-memory cache is empty (e.g. just restarted).
func (h *SyncHandler) PublicLatestStatus(c *gin.Context) {
	if h.housingSync != nil {
		if last := h.housingSync.LastResult(); last != nil {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "result": last})
			return
		}
	}

	last, err := h.repo.LatestSyncResult(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			errorResponse("DB_ERROR", "failed to query sync history"))
		return
	}
	if last == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "no sync has been executed yet",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "result": last})
}

// PublicHistory returns the last N sync results. ?limit=N (default 10, max 100).
func (h *SyncHandler) PublicHistory(c *gin.Context) {
	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	results, err := h.repo.RecentSyncHistory(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			errorResponse("DB_ERROR", "failed to query sync history"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"count":   len(results),
		"results": results,
	})
}

func (h *SyncHandler) authorize(c *gin.Context) bool {
	if h.adminKey == "" {
		c.JSON(http.StatusServiceUnavailable, errorResponse("CONFIG_ERROR", "admin key not configured"))
		return false
	}
	if c.GetHeader("X-Admin-Key") != h.adminKey {
		c.JSON(http.StatusUnauthorized, errorResponse("UNAUTHORIZED", "invalid admin key"))
		return false
	}
	return true
}
