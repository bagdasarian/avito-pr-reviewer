package server

import (
	"context"
	"log"
	"net/http"

	"github.com/bagdasarian/avito-pr-reviewer/internal/handler"
)

type Server struct {
	handler *handler.Handler
	server  *http.Server
}

func NewServer(h *handler.Handler, addr string) *Server {
	mux := http.NewServeMux()
	SetupRoutes(mux, h)

	return &Server{
		handler: h,
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

func (s *Server) Start() error {
	log.Printf("Server starting on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down...")
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}
	log.Println("Server stopped")
	return nil
}
