package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"kr-metro-api/repository"
)

type LineHandler struct {
	repo repository.LineRepository
}

func NewLineHandler(repo repository.LineRepository) *LineHandler {
	return &LineHandler{repo: repo}
}

func (h *LineHandler) List(c *gin.Context) {
	lines, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	c.JSON(http.StatusOK, lines)
}

func (h *LineHandler) ListStations(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "line id must be an integer"))
		return
	}
	result, err := h.repo.ListStations(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	if result == nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "line not found or has no stations"))
		return
	}
	c.Data(http.StatusOK, "application/geo+json", result)
}

func (h *LineHandler) GetGeometry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "line id must be an integer"))
		return
	}
	result, err := h.repo.GetGeometry(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	if result == nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "line geometry not found"))
		return
	}
	c.Data(http.StatusOK, "application/geo+json", result)
}
