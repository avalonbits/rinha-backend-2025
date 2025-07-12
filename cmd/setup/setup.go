package setup

import (
	"sync/atomic"

	"github.com/avalonbits/rinha-backend-2025/config"
	"github.com/labstack/echo/v4"
)

func Echo(cfg config.Config) *Server {
	server := &Server{}
	server.healthy.Store(true)

	e := echo.New()
	server.Echo = e

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
