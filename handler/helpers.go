package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func parsePagination(c *gin.Context) (page, perPage int) {
	page = 1
	perPage = 50
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := c.Query("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp > 0 && pp <= 200 {
			perPage = pp
		}
	}
	return
}

func setPaginationHeaders(c *gin.Context, total, page, perPage int) {
	c.Header("X-Total-Count", strconv.Itoa(total))
	c.Header("X-Page", strconv.Itoa(page))
	c.Header("X-Per-Page", strconv.Itoa(perPage))
}

func errorResponse(code, message string) gin.H {
	return gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	}
}
