package plot

import (
	"fmt"
	"net/http"
)

// HTTPServer defines the interface for an HTTP server that Chart will use
type HTTPServer interface {
	// RegisterHandler registers a handler for a specific route
	RegisterHandler(path string, handler http.HandlerFunc)

	// RegisterFileServer registers a handler to serve static files
	RegisterFileServer(path string, fs http.FileSystem)

	// Start starts the HTTP server on the specified port
	Start(port int) error
}

// StandardHTTPServer implements the HTTPServer interface using the standard http package
type StandardHTTPServer struct{}

// NewStandardHTTPServer creates a new instance of StandardHTTPServer
func NewStandardHTTPServer() *StandardHTTPServer {
	return &StandardHTTPServer{}
}

// RegisterHandler registers a handler for a specific route
func (s *StandardHTTPServer) RegisterHandler(path string, handler http.HandlerFunc) {
	http.HandleFunc(path, handler)
}

// RegisterFileServer registers a handler to serve static files
func (s *StandardHTTPServer) RegisterFileServer(path string, fs http.FileSystem) {
	http.Handle(path, http.FileServer(fs))
}

// Start starts the HTTP server on the specified port
func (s *StandardHTTPServer) Start(port int) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
