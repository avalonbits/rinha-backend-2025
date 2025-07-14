package setup

import (
	"sync/atomic"

	"github.com/avalonbits/rinha-backend-2025/config"
	"github.com/avalonbits/rinha-backend-2025/endpoints/api"
	"github.com/avalonbits/rinha-backend-2025/service/payment"
	"github.com/avalonbits/rinha-backend-2025/storage/sharded"
	"github.com/labstack/echo/v4"
)

func Echo(cfg config.Config) *Server {
	server := &Server{}
	server.healthy.Store(true)

	e := echo.New()
	server.Echo = e

	store := sharded.New(64, cfg.Database)
	payments := payment.New(cfg.PaymentProcessorDefault, cfg.PaymentProcessorBackup, store)
	handlers := api.New(payments)
	e.POST("/payments", handlers.ProcessPayment)
	e.GET("/payments-summary", handlers.PaymentSummary)

	return server
}

type Server struct {
	*echo.Echo

	healthy atomic.Bool
	// db      *datastore.DB
}

func (s *Server) MarkUnavailable() {
	s.healthy.Store(false)
}

func (s *Server) Cleanup() {
	// s.db.Close()
}
