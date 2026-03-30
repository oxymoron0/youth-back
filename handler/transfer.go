package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"kr-metro-api/repository"
)

type TransferHandler struct {
	repo repository.TransferRepository
}

func NewTransferHandler(repo repository.TransferRepository) *TransferHandler {
	return &TransferHandler{repo: repo}
}

func (h *TransferHandler) GetByStation(c *gin.Context) {
	stationID, err := strconv.Atoi(c.Param("station_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "station_id must be an integer"))
		return
	}
	transfers, err := h.repo.GetByStation(c.Request.Context(), stationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	c.JSON(http.StatusOK, transfers)
}
