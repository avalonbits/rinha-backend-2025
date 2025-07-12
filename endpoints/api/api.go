package api

import (
	"net/http"

	"github.com/avalonbits/rinha-backend-2025/service/payment"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	payments *payment.Service
}

func New(payments *payment.Service) *Handler {
	return &Handler{
		payments: payments,
	}
}

type processPaymentRequest struct {
	CorrleationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

func (h *Handler) ProcessPayment(c echo.Context) error {
	return c.String(http.StatusOK, "")
}

type paymentSummaryResponse struct {
	Default paymentProcessor `json:"defalt"`
	Backup  paymentProcessor `json:"fallback"`
}

type paymentProcessor struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

func (h *Handler) PaymentSummary(c echo.Context) error {
	return c.JSON(http.StatusOK, paymentSummaryResponse{})
}
