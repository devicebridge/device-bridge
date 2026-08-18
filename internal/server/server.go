package server

import "net/http"

const (
	HealthPath = "/healthz"
	ReadyPath  = "/readyz"
)

// Server provides HTTP routing.
type Server struct {
	mux *http.ServeMux
}

// New creates a new HTTP server.
func New() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}
	s.Handle(HealthPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	s.Handle(ReadyPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	return s
}

// Handle registers an HTTP handler.
func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}
