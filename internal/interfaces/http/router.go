package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "centropy-affilate/docs"
	"centropy-affilate/internal/interfaces/http/handler"
	appmiddleware "centropy-affilate/internal/interfaces/http/middleware"
	"centropy-affilate/pkg/cqrs"
)

type Deps struct {
	Bus            *cqrs.Bus
	Logger         *slog.Logger
	TokenParser    appmiddleware.TokenParser
	AllowedOrigins []string
	Env            string
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(appmiddleware.Recover(deps.Logger))
	r.Use(appmiddleware.Logging(deps.Logger))
	// No blanket request timeout here — the two LLM-batch trigger routes
	// (analysis/run, complaints/.../verify) need much longer than any other
	// route, and chi's Timeout composes as "earliest deadline wins", so a
	// looser one set closer to those routes can't override a tighter one
	// set here. Each route group below sets its own.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   deps.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", handler.Health)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	authHandler := handler.NewAuthHandler(deps.Bus)
	segmentHandler := handler.NewSegmentHandler(deps.Bus)
	customerHandler := handler.NewCustomerHandler(deps.Bus)
	complaintHandler := handler.NewComplaintHandler(deps.Bus)
	renewalHandler := handler.NewRenewalHandler(deps.Bus)
	analysisHandler := handler.NewAnalysisHandler(deps.Bus)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(deps.TokenParser))
			r.Use(chimiddleware.Timeout(30 * time.Second))

			r.Route("/admin/segments", func(r chi.Router) {
				r.Get("/", segmentHandler.Summary)
				r.Get("/non-purchasers", segmentHandler.ListNonPurchasers)
				r.Get("/non-purchasers/monthly", segmentHandler.MonthlyNonPurchaserSignups)
				r.Get("/{segment}/customers", segmentHandler.ListCustomers)
			})

			r.Route("/admin/customers", func(r chi.Router) {
				r.Get("/", customerHandler.List)
				r.Post("/sync", customerHandler.Sync)
			})

			r.Route("/admin/complaints", func(r chi.Router) {
				r.Get("/delayed-program", complaintHandler.ListDelayedProgramComplainers)
				r.Get("/delayed-program/verified", complaintHandler.ListVerified)
			})

			r.Route("/admin/renewals", func(r chi.Router) {
				r.Get("/overdue", renewalHandler.ListOverdue)
			})

			r.Route("/admin/analysis", func(r chi.Router) {
				r.Get("/overdue", analysisHandler.ListOverdue)
			})
		})

		// LLM-batch trigger routes: up to maxManualRunLimit/maxManualVerifyLimit
		// sequential GapGPT calls in one request (a few seconds each), so
		// these need a much longer budget than every other route.
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(deps.TokenParser))
			r.Use(chimiddleware.Timeout(4*time.Minute + 30*time.Second))

			r.Post("/admin/complaints/delayed-program/verify", complaintHandler.Verify)
			r.Post("/admin/analysis/run", analysisHandler.Run)
		})
	})

	return r
}
