package server

import "net/http"

// Server provides HTTP routing.
type Server struct {
	mux *http.ServeMux
}

// New creates a new HTTP server.
func New() *Server {
	return &Server{
		mux: http.NewServeMux(),
	}
}

// Handle registers an HTTP handler.
func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}
