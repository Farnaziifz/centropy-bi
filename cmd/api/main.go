// Command api boots the centropy-affilate admin API: loads config, opens
// this service's own Postgres (via ent), Redis, and a read-only connection
// to the AlefGym production database, wires every domain repository and
// CQRS handler, and serves until an interrupt/SIGTERM triggers a graceful
// shutdown.
//
// This service is the backend for the loyalty-club roadmap
// (loyalty-club-roadmap.html): it classifies every AlefGym customer into
// one of six segments (newcomer/cold/hero/at-risk/churned/one-time) and
// exposes that to an internal admin dashboard. It owns no customer PII
// beyond what it syncs read-only from AlefGym, and never writes back to
// AlefGym.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	analysiscmd "centropy-affilate/internal/application/analysis/command"
	analysisquery "centropy-affilate/internal/application/analysis/query"
	authcmd "centropy-affilate/internal/application/auth/command"
	complaintcmd "centropy-affilate/internal/application/complaint/command"
	complaintquery "centropy-affilate/internal/application/complaint/query"
	customercmd "centropy-affilate/internal/application/customer/command"
	customerquery "centropy-affilate/internal/application/customer/query"
	renewalquery "centropy-affilate/internal/application/renewal/query"
	segmentquery "centropy-affilate/internal/application/segment/query"
	"centropy-affilate/internal/domain/complaint"
	"centropy-affilate/internal/domain/customer"
	"centropy-affilate/internal/domain/renewal"
	"centropy-affilate/internal/domain/segment"
	"centropy-affilate/internal/infrastructure/alefgym"
	infraauth "centropy-affilate/internal/infrastructure/auth"
	"centropy-affilate/internal/infrastructure/cache"
	"centropy-affilate/internal/infrastructure/config"
	"centropy-affilate/internal/infrastructure/gapgpt"
	"centropy-affilate/internal/infrastructure/logger"
	"centropy-affilate/internal/infrastructure/persistence"
	transporthttp "centropy-affilate/internal/interfaces/http"
	"centropy-affilate/pkg/cqrs"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.Env)

	entClient, err := persistence.NewEntClient(cfg.DB)
	if err != nil {
		return err
	}
	defer entClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := persistence.RunMigrations(ctx, entClient); err != nil {
		cancel()
		return err
	}
	if err := persistence.SeedDefaultAdmin(ctx, entClient, os.Getenv("ADMIN_SEED_EMAIL"), os.Getenv("ADMIN_SEED_PASSWORD")); err != nil {
		cancel()
		return err
	}
	cancel()

	alefgymDB, err := alefgym.NewClient(cfg.AlefGym.DSN)
	if err != nil {
		return err
	}
	defer alefgymDB.Close()

	redisCache := cache.New(cfg.Redis)
	defer redisCache.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisCache.Ping(pingCtx); err != nil {
		log.Warn("redis unavailable at startup, query caching disabled until it recovers", "error", err)
	}
	pingCancel()

	// --- repositories & adapters -------------------------------------------------
	adminUserRepo := persistence.NewAdminUserRepository(entClient)
	customerRepo := persistence.NewCustomerRepository(entClient)
	customerSource := alefgym.NewCustomerSource(alefgymDB, cfg.AlefGym.ExcludedUserIDs)
	segmentRepo := alefgym.NewSegmentRepository(alefgymDB, cfg.AlefGym.ExcludedUserIDs, log)
	complaintRepo := alefgym.NewComplaintRepository(alefgymDB, cfg.AlefGym.ExcludedUserIDs)
	renewalRepo := alefgym.NewRenewalRepository(alefgymDB, cfg.AlefGym.ExcludedUserIDs)
	messageSource := alefgym.NewMessageSource(alefgymDB)
	analysisRepo := persistence.NewAnalysisRepository(entClient)
	verificationRepo := persistence.NewComplaintVerificationRepository(entClient)
	gapgptClient := gapgpt.NewClient(cfg.GapGPT.BaseURL, cfg.GapGPT.APIKey, cfg.GapGPT.Model)

	jwtService := infraauth.NewJWTService(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL)

	// --- CQRS bus & handler registration ----------------------------------------
	bus := cqrs.NewBus()

	loginHandler := authcmd.NewLoginHandler(adminUserRepo, jwtService, infraauth.VerifyPassword)
	cqrs.RegisterCommand[authcmd.LoginCommand, authcmd.LoginResult](bus, loginHandler.Handle)

	syncCustomersHandler := customercmd.NewSyncCustomersHandler(customerSource, customerRepo)
	cqrs.RegisterCommand[customercmd.SyncCustomersCommand, customercmd.SyncCustomersResult](bus, syncCustomersHandler.Handle)

	listCustomersHandler := customerquery.NewListCustomersHandler(customerRepo)
	cqrs.RegisterQuery[customerquery.ListCustomersQuery, []customer.Customer](bus, listCustomersHandler.Handle)

	getSummaryHandler := segmentquery.NewGetSummaryHandler(segmentRepo, redisCache)
	cqrs.RegisterQuery[segmentquery.GetSummaryQuery, segment.Summary](bus, getSummaryHandler.Handle)

	listSegmentCustomersHandler := segmentquery.NewListCustomersHandler(segmentRepo, redisCache)
	cqrs.RegisterQuery[segmentquery.ListCustomersQuery, []segment.Customer](bus, listSegmentCustomersHandler.Handle)

	listNonPurchasersHandler := segmentquery.NewListNonPurchasersHandler(segmentRepo, redisCache)
	cqrs.RegisterQuery[segmentquery.ListNonPurchasersQuery, []segment.NonPurchaser](bus, listNonPurchasersHandler.Handle)

	monthlyNonPurchasersHandler := segmentquery.NewMonthlyNonPurchaserSignupsHandler(segmentRepo, redisCache)
	cqrs.RegisterQuery[segmentquery.MonthlyNonPurchaserSignupsQuery, []segment.MonthlySignups](bus, monthlyNonPurchasersHandler.Handle)

	listDelayedProgramComplainersHandler := complaintquery.NewListDelayedProgramComplainersHandler(complaintRepo, redisCache)
	cqrs.RegisterQuery[complaintquery.ListDelayedProgramComplainersQuery, []complaint.DelayedProgramComplainer](bus, listDelayedProgramComplainersHandler.Handle)

	verifyDelayedComplaintsHandler := complaintcmd.NewVerifyDelayedComplaintsHandler(complaintRepo, verificationRepo, gapgptClient, log)
	cqrs.RegisterCommand[complaintcmd.VerifyDelayedComplaintsCommand, complaintcmd.VerifyDelayedComplaintsResult](bus, verifyDelayedComplaintsHandler.Handle)

	listVerifiedComplainersHandler := complaintquery.NewListVerifiedComplainersHandler(complaintRepo, verificationRepo)
	cqrs.RegisterQuery[complaintquery.ListVerifiedComplainersQuery, []complaintquery.VerifiedComplainer](bus, listVerifiedComplainersHandler.Handle)

	listOverdueHandler := renewalquery.NewListOverdueHandler(renewalRepo, redisCache)
	cqrs.RegisterQuery[renewalquery.ListOverdueQuery, []renewal.OverdueCustomer](bus, listOverdueHandler.Handle)

	runDailyAnalysisHandler := analysiscmd.NewRunDailyAnalysisHandler(renewalRepo, analysisRepo, messageSource, gapgptClient, log)
	cqrs.RegisterCommand[analysiscmd.RunDailyAnalysisCommand, analysiscmd.RunDailyAnalysisResult](bus, runDailyAnalysisHandler.Handle)

	listOverdueWithAnalysisHandler := analysisquery.NewListOverdueWithAnalysisHandler(renewalRepo, analysisRepo)
	cqrs.RegisterQuery[analysisquery.ListOverdueWithAnalysisQuery, []analysisquery.OverdueWithAnalysis](bus, listOverdueWithAnalysisHandler.Handle)

	// --- daily analysis scheduler ------------------------------------------------
	// A plain in-process ticker, not a cron library: these jobs have exactly
	// one schedule, ever, so a dependency for cron expression parsing would
	// be pure overhead. RunOnStartup is off by default so a dev `go run`
	// never silently spends GapGPT credits — see AnalysisConfig's doc comment.
	go func() {
		runOnce := func() {
			analysisResult, err := cqrs.ExecuteCommand[analysiscmd.RunDailyAnalysisCommand, analysiscmd.RunDailyAnalysisResult](
				context.Background(), bus, analysiscmd.RunDailyAnalysisCommand{Limit: 0},
			)
			if err != nil {
				log.Error("daily analysis job failed", "error", err)
			} else {
				log.Info("daily analysis job finished",
					"candidates", analysisResult.Candidates, "analyzed", analysisResult.Analyzed,
					"skipped", analysisResult.Skipped, "failed", analysisResult.Failed)
			}

			verifyResult, err := cqrs.ExecuteCommand[complaintcmd.VerifyDelayedComplaintsCommand, complaintcmd.VerifyDelayedComplaintsResult](
				context.Background(), bus, complaintcmd.VerifyDelayedComplaintsCommand{Limit: 0},
			)
			if err != nil {
				log.Error("daily complaint verification job failed", "error", err)
			} else {
				log.Info("daily complaint verification job finished",
					"candidates", verifyResult.Candidates, "verified", verifyResult.Verified,
					"skipped", verifyResult.Skipped, "failed", verifyResult.Failed)
			}
		}

		if cfg.Analysis.RunOnStartup {
			runOnce()
		}
		ticker := time.NewTicker(cfg.Analysis.RunInterval)
		defer ticker.Stop()
		for range ticker.C {
			runOnce()
		}
	}()

	// --- HTTP server --------------------------------------------------------------
	router := transporthttp.NewRouter(transporthttp.Deps{
		Bus:            bus,
		Logger:         log,
		TokenParser:    jwtService,
		AllowedOrigins: cfg.HTTP.AllowedOrigins,
		Env:            cfg.Env,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "port", cfg.HTTP.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop, stopCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopCancel()

	select {
	case err := <-errCh:
		return err
	case <-stop.Done():
		log.Info("shutting down")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()
	return srv.Shutdown(shutdownCtx)
}
