package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"example.com/disbursement/internal/models"
	"example.com/disbursement/internal/services"
)

func (h *Handler) CreateDisbursement(c *gin.Context) {
	var req models.CreateDisbursementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "request body tidak valid",
		})
		return
	}

	username := c.GetString("username")
	d, err := h.service.Create(req, username)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Disbursement berhasil dibuat",
		"data":    d,
	})
}

func (h *Handler) UpdateDisbursementStatus(c *gin.Context) {
	role := c.GetString("role")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "hanya admin yang dapat mengubah status disbursement",
		})
		return
	}

	id := c.Param("id")

	var req models.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "request body tidak valid",
		})
		return
	}

	username := c.GetString("username")
	d, err := h.service.UpdateStatus(id, req, username)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if err == services.ErrNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Status disbursement berhasil diperbarui",
		"data":    d,
	})
}

func (h *Handler) ListDisbursements(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	params := models.ListDisbursementParams{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
		Status: c.Query("status"),
	}

	data, total, err := h.service.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "terjadi kesalahan pada server",
		})
		return
	}

	totalPages := int(total) / limit
	if limit > 0 && int(total)%limit != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}
