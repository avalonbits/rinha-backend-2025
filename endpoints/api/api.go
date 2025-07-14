package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/avalonbits/rinha-backend-2025/service/payment"
	"github.com/google/uuid"
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
	CorrelationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

func (r *processPaymentRequest) validate(c echo.Context) error {
	r.CorrelationID = strings.TrimSpace(r.CorrelationID)
	if r.CorrelationID == "" {
		return fmt.Errorf("correlationID is required")
	}
	if _, err := uuid.Parse(r.CorrelationID); err != nil {
		return fmt.Errorf("invalid uuid")
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

	processor, err := h.payments.ProcessPayment(ctx, r.CorrelationID, r.Amount, createdAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	if err := h.payments.Log(ctx, r.CorrelationID, r.Amount, createdAt, processor); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, "")
}

type paymentSummaryRequest struct {
	From string `query:"from"`
	To   string `query:"to"`
}

func (r *paymentSummaryRequest) validate(c echo.Context) error {
	r.From = strings.TrimSpace(r.From)
	if r.From != "" {
		if _, err := time.Parse(time.RFC3339Nano, r.From); err != nil {
			return fmt.Errorf("invalid time fromat in 'from'")
		}
	}

	r.To = strings.TrimSpace(r.To)
	if r.To == "" {
		r.To = time.Now().UTC().Format(time.RFC3339Nano)
	} else if _, err := time.Parse(time.RFC3339Nano, r.To); err != nil {
		return fmt.Errorf("invalid time fromat in 'to'")
	}

	return nil
}

func (h *Handler) PaymentSummary(c echo.Context) error {
	r := paymentSummaryRequest{}
	if err := h.validateRequest(c, &r); err != nil {
		return err
	}

	ctx := c.Request().Context()
	summary, err := h.payments.Summary(ctx, r.From, r.To)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, summary)
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
