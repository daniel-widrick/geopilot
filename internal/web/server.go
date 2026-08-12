// Package web serves the dashboard and pushes the live decoded model to the
// browser over a WebSocket (no polling, no page refresh).
package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	_ "embed"

	"github.com/daniel-widrick/geopilot/internal/live"
	"github.com/daniel-widrick/geopilot/internal/model"
	"github.com/daniel-widrick/geopilot/internal/store"
	"github.com/gorilla/websocket"
)

//go:embed dashboard.html
var dashboardHTML []byte

type Server struct {
	snap    *live.Snapshot
	db      *store.Store
	upgrade websocket.Upgrader
	push    time.Duration

	mu        sync.RWMutex
	rate      float64  // cached current $/kWh
	costToday *float64 // cached spend so far today
}

// refreshCost periodically caches the effective rate and today's cost so each
// WebSocket frame doesn't re-hit the database.
func (s *Server) refreshCost(ctx context.Context) {
	tick := func() {
		rate, _ := s.db.CurrentRate(ctx)
		ct, ok, _ := s.db.CostToday(ctx, rate)
		s.mu.Lock()
		s.rate = rate
		if ok {
			s.costToday = &ct
		}
		s.mu.Unlock()
	}
	tick()
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tick()
		}
	}
}

func New(snap *live.Snapshot, db *store.Store, push time.Duration) *Server {
	if push <= 0 {
		push = 2 * time.Second
	}
	return &Server{snap: snap, db: db, push: push,
		upgrade: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(dashboardHTML)
	})
	mux.HandleFunc("/ws", s.serveWS)
	mux.HandleFunc("/api/history", s.serveHistory)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	return mux
}

func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrade.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	t := time.NewTicker(s.push)
	defer t.Stop()
	// send one immediately so the page paints without waiting a tick
	if err := s.sendModel(conn); err != nil {
		return
	}
	for range t.C {
		if err := s.sendModel(conn); err != nil {
			return
		}
	}
}

func (s *Server) sendModel(conn *websocket.Conn) error {
	s.mu.RLock()
	rate, ct := s.rate, s.costToday
	s.mu.RUnlock()
	m := model.Build(s.snap, time.Now().Format(time.RFC3339), rate)
	m.CostToday = ct
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, b)
}

// serveHistory returns 1-minute rollup points for a series key over ?hours=N.
func (s *Server) serveHistory(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("series")
	if key == "" {
		http.Error(w, "series required", http.StatusBadRequest)
		return
	}
	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if n, err := time.ParseDuration(h + "h"); err == nil {
			hours = int(n.Hours())
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	pts, err := s.db.History(ctx, key, hours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pts)
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	go s.refreshCost(ctx)
	srv := &http.Server{Addr: addr, Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		sh, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(sh)
	}()
	log.Printf("web: listening on %s", addr)
	return srv.ListenAndServe()
}
