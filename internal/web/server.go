// Package web serves the embedded admin console: a single-page config UI plus
// a small JSON API for managing peers, testing connections and inspecting
// bridge status. It binds to a loopback address by default and is kept separate
// from the MCP HTTP endpoint.
package web

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/medcelerate/MOS-MCP/internal/config"
	"github.com/medcelerate/MOS-MCP/internal/mos"
)

//go:embed static/*
var staticFS embed.FS

// Server is the admin console HTTP server.
type Server struct {
	mgr     *mos.Manager
	cfgPath string
	logf    func(string, ...any)

	mu  sync.Mutex // guards cfg mutations + persistence
	cfg *config.Config
}

// New creates an admin server. cfgPath may be empty, in which case edits apply
// in memory but are not persisted to disk.
func New(mgr *mos.Manager, cfg *config.Config, cfgPath string, logf func(string, ...any)) *Server {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Server{mgr: mgr, cfg: cfg, cfgPath: cfgPath, logf: logf}
}

// Handler returns the HTTP handler serving the console and API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	sub, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(sub)))

	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/peers", s.handlePeers)
	mux.HandleFunc("/api/peers/test", s.handleTestPeer)
	mux.HandleFunc("/api/inbox", s.handleInbox)

	return mux
}

// Serve runs the admin server on addr until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	s.logf("admin console listening on http://%s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
