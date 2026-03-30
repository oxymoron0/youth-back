package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"kr-metro-api/repository"
)

type HousingHandler struct {
	repo repository.HousingRepository
}

func NewHousingHandler(repo repository.HousingRepository) *HousingHandler {
	return &HousingHandler{repo: repo}
}

func (h *HousingHandler) List(c *gin.Context) {
	items, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *HousingHandler) GetByHomeCode(c *gin.Context) {
	homeCode := c.Param("home_code")
	detail, err := h.repo.GetByHomeCode(c.Request.Context(), homeCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "housing not found"))
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *HousingHandler) NearbyStations(c *gin.Context) {
	homeCode := c.Param("home_code")

	// Verify housing exists
	detail, err := h.repo.GetByHomeCode(c.Request.Context(), homeCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "housing not found"))
		return
	}

	distance := 150
	if v := c.Query("distance"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 && d <= 50000 {
			distance = d
		}
	}

	stations, err := h.repo.NearbyStations(c.Request.Context(), homeCode, distance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	c.JSON(http.StatusOK, stations)
}
