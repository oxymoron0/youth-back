package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kr-metro-api/sync"
)

type SyncHandler struct {
	housingSync *sync.HousingSync
	adminKey    string
}

func NewSyncHandler(housingSync *sync.HousingSync, adminKey string) *SyncHandler {
	return &SyncHandler{housingSync: housingSync, adminKey: adminKey}
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
