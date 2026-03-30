package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"kr-metro-api/repository"
)

type StationHandler struct {
	repo repository.StationRepository
}

func NewStationHandler(repo repository.StationRepository) *StationHandler {
	return &StationHandler{repo: repo}
}

func (h *StationHandler) List(c *gin.Context) {
	page, perPage := parsePagination(c)

	accept := c.GetHeader("Accept")
	format := c.Query("format")
	wantGeoJSON := strings.Contains(accept, "geo+json") || format == "geojson"

	if wantGeoJSON {
		lineIDStr := c.Query("line_id")
		var lineID *int
		if lineIDStr != "" {
			if lid, err := strconv.Atoi(lineIDStr); err == nil {
				lineID = &lid
			}
		}
		data, err := h.repo.ListGeoJSONAll(c.Request.Context(), lineID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
			return
		}
		c.Data(http.StatusOK, "application/geo+json", data)
		return
	}

	stations, total, err := h.repo.List(c.Request.Context(), page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	setPaginationHeaders(c, total, page, perPage)
	c.JSON(http.StatusOK, stations)
}

func (h *StationHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "station id must be an integer"))
		return
	}
	detail, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "station not found"))
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *StationHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if len(q) < 1 {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_QUERY", "query must be at least 1 character"))
		return
	}
	limit := 20
	if v := c.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	results, err := h.repo.Search(c.Request.Context(), q, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	c.JSON(http.StatusOK, results)
}

func (h *StationHandler) Nearby(c *gin.Context) {
	lonStr := c.Query("lon")
	latStr := c.Query("lat")
	if lonStr == "" || latStr == "" {
		c.JSON(http.StatusBadRequest, errorResponse("MISSING_PARAMS", "lon and lat are required"))
		return
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_PARAM", "lon must be a number"))
		return
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_PARAM", "lat must be a number"))
		return
	}
	radius := 1000
	if v := c.Query("radius"); v != "" {
		if r, err := strconv.Atoi(v); err == nil && r > 0 && r <= 50000 {
			radius = r
		}
	}
	limit := 20
	if v := c.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	result, err := h.repo.Nearby(c.Request.Context(), lon, lat, radius, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL", err.Error()))
		return
	}
	c.Data(http.StatusOK, "application/geo+json", result)
}
