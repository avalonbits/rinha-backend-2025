package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

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

func (r *processPaymentRequest) validate(c echo.Context) error {
	r.CorrleationID = strings.TrimSpace(r.CorrleationID)
	if r.CorrleationID == "" {
		return fmt.Errorf("correlationID is required")
	}

	if r.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	return nil
}

func (h *Handler) ProcessPayment(c echo.Context) error {
	r := processPaymentRequest{}
	if err := h.validateRequest(c, &r); err != nil {
		return err
	}

	ctx := c.Request().Context()
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	client, err := h.getClient()
	if err != nil {
		return err
	}

	if err := client.ProcessPayment(ctx, r.CorrleationID, r.Amount, createdAt); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

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

type validator interface {
	validate(echo.Context) error
}

func (h *Handler) validateRequest(c echo.Context, req validator) error {
	var err error
	if err = c.Bind(req); err == nil {
		err = req.validate(c)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func (h *Handler) getClient() (*payment.Client, error) {
	client := h.payments.PreferDefault()
	if client == nil {
		return nil, echo.NewHTTPError(
			http.StatusServiceUnavailable,
			"payment process is currently unavailable",
		)
	}

	return client, nil
}
