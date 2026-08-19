package handlers

import "example.com/disbursement/internal/services"

type Handler struct {
	service *services.DisbursementService
}

func NewHandler(service *services.DisbursementService) *Handler {
	return &Handler{service: service}
}
