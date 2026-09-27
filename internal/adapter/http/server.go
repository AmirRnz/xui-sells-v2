package http

import (
	"context"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"xui-sells-v2/internal/app/stats"
	"xui-sells-v2/internal/app/ticket"
	"xui-sells-v2/internal/app/wallet"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/web"
)

// ResellerAccount represents an authenticated reseller credentials record.
type ResellerAccount struct {
	TelegramID    int64
	InstanceID    int64
	UserID        int64
	Username      string
	FirstName     string
	PasswordHash  string
	Tier          domain.ResellerTier
	TierExpiresAt *time.Time
	ServiceName   string
}

// ResellerStore abstracts reseller credentials lookup and password updating.
type ResellerStore interface {
	GetAccountByTgID(ctx context.Context, tgID int64) (*ResellerAccount, error)
	UpdatePasswordHash(ctx context.Context, tgID int64, newHash string) error
}

// ServiceStore abstracts active subscription queries and panel operations for resellers.
type ServiceStore interface {
	ListResellerServices(ctx context.Context, instanceID int64, search string, status string) ([]domain.Service, error)
	GetServiceByID(ctx context.Context, id int64) (*domain.Service, error)
	GetServiceByEmail(ctx context.Context, email string) (*domain.Service, error)
	ResetTraffic(ctx context.Context, idOrEmail string) error
	RotateSubID(ctx context.Context, serviceID int64) (*domain.Service, string, error)
}

// OrderStore abstracts order inspection and decision actions for resellers.
type OrderStore interface {
	ListOrders(ctx context.Context, instanceID int64, status string) ([]domain.Order, error)
	ApproveOrder(ctx context.Context, orderID int64) error
	RejectOrder(ctx context.Context, orderID int64, reason string) error
}

// ChildBotStore abstracts child bot queries under a reseller.
type ChildBotStore interface {
	ListChildBots(ctx context.Context, parentInstanceID int64) ([]domain.Instance, error)
}

// Server provides the HTTP router, REST API, and static SPA server.
type Server struct {
	router        chi.Router
	sessions      *SessionManager
	resellerStore ResellerStore
	serviceStore  ServiceStore
	orderStore    OrderStore
	childBotStore ChildBotStore
	walletRepo    wallet.WalletRepository
	statsSvc      *stats.Service
	ticketSvc     *ticket.Service
	webFS         fs.FS
}

// Config configures dependencies for the HTTP server.
type Config struct {
	ResellerStore ResellerStore
	ServiceStore  ServiceStore
	OrderStore    OrderStore
	ChildBotStore ChildBotStore
	WalletRepo    wallet.WalletRepository
	StatsSvc      *stats.Service
	TicketSvc     *ticket.Service
	WebFS         fs.FS
}

// NewServer builds and configures the HTTP router.
func NewServer(cfg Config) *Server {
	s := &Server{
		router:        chi.NewRouter(),
		sessions:      NewSessionManager(),
		resellerStore: cfg.ResellerStore,
		serviceStore:  cfg.ServiceStore,
		orderStore:    cfg.OrderStore,
		childBotStore: cfg.ChildBotStore,
		walletRepo:    cfg.WalletRepo,
		statsSvc:      cfg.StatsSvc,
		ticketSvc:     cfg.TicketSvc,
		webFS:         cfg.WebFS,
	}

	if s.webFS == nil {
		s.webFS = web.DistFS
	}
	if sub, err := fs.Sub(s.webFS, "dist"); err == nil {
		if _, err := fs.Stat(sub, "index.html"); err == nil {
			s.webFS = sub
		}
	}

	s.setupRoutes()
	return s
}

// Router returns the chi router for mounting or testing.
func (s *Server) Router() chi.Router {
	return s.router
}

// Handler returns the http.Handler for serving traffic.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) setupRoutes() {
	r := s.router

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Liveness & health check probes
	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","version":"2.0.0"}`))
	})
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","version":"2.0.0"}`))
	})

	// API route registration helper
	registerAPIRoutes := func(api chi.Router) {
		// Public Auth routes
		api.Post("/login", s.handleLogin)
		api.Post("/auth/login", s.handleLogin)

		// Protected routes
		api.Group(func(protected chi.Router) {
			protected.Use(s.AuthMiddleware)

			// Session Profile & Auth
			protected.Get("/me", s.handleGetMe)
			protected.Get("/auth/me", s.handleGetMe)
			protected.Post("/logout", s.handleLogout)
			protected.Post("/auth/logout", s.handleLogout)
			protected.Post("/change-password", s.handleChangePassword)
			protected.Post("/auth/change-password", s.handleChangePassword)

			// Web Panel Gated Endpoints (requires Ultimate tier per D07)
			protected.Group(func(gated chi.Router) {
				gated.Use(s.RequireWebPanelMiddleware)

				// Stats
				gated.Get("/stats", s.handleGetStats)

				// Services
				gated.Get("/services", s.handleGetServices)
				gated.Post("/services/{id}/reset-traffic", s.handleResetTraffic)
				gated.Post("/services/{id}/rotate-sub", s.handleRotateSubID)
				gated.Post("/services/{id}/rotate-subid", s.handleRotateSubID)

				// Orders
				gated.Get("/orders", s.handleGetOrders)
				gated.Post("/orders/{id}/approve", s.handleApproveOrder)
				gated.Post("/orders/{id}/reject", s.handleRejectOrder)

				// Child Bots
				gated.Get("/child-bots", s.handleGetChildBots)

				// Tickets
				gated.Get("/tickets", s.handleGetTickets)
				gated.Post("/tickets/{id}/reply", s.handleReplyTicket)
			})
		})
	}

	// Mount at both /api/reseller (used by web/src/api/client.ts) and /api
	r.Route("/api/reseller", registerAPIRoutes)
	r.Route("/api", registerAPIRoutes)

	// Serve Static Files & SPA fallback from web/dist
	if s.webFS != nil {
		fileServer := http.FileServer(http.FS(s.webFS))

		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			path := strings.TrimPrefix(req.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			// Check if file physically exists in dist
			if f, err := s.webFS.Open(path); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, req)
				return
			}

			// SPA fallback: return index.html for client-side routing
			indexData, err := fs.ReadFile(s.webFS, "index.html")
			if err != nil {
				http.NotFound(w, req)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(indexData)
		})
	}
}
