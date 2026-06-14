package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-co-op/gocron/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/api"
	"github.com/maximseshuk/snapr/internal/backup"
	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/i18n"
	"github.com/maximseshuk/snapr/internal/logger"
	"github.com/maximseshuk/snapr/internal/utils"
)

const shutdownTimeout = 10 * time.Second

func Run(configPath, version string) error {
	logger.SetGlobalLevel(logger.GetLevel())
	// Stdout-only until config tells us where to write log files.
	logger.Setup(nil)

	systemLogger := logger.NewSystemLogger("main")
	systemLogger.Info().Msg("Starting snapr")

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	sinks, err := logger.NewFileSinks(cfg.Logs)
	if err != nil {
		return fmt.Errorf("init log files: %w", err)
	}
	defer sinks.Close()
	logger.Setup(sinks)
	if sinks != nil {
		systemLogger.Info().
			Str("path", cfg.Logs.Path).
			Bool("system", cfg.Logs.System).
			Bool("per_job", cfg.Logs.PerJob).
			Msg("File logs enabled")
	}

	serverEnabled := cfg.Server.Enabled
	uiEnabled := serverEnabled && cfg.Server.UI.Enabled

	if serverEnabled {
		if err := i18n.Init(cfg.Server.DefaultLanguage); err != nil {
			return fmt.Errorf("init i18n: %w", err)
		}
		systemLogger.Debug().Str("language", cfg.Server.DefaultLanguage).Msg("i18n initialized")
	} else {
		systemLogger.Info().Msg("HTTP server disabled, running scheduler only")
	}

	backupManager := backup.NewManager(cfg)

	scheduler, err := startScheduler(cfg, backupManager)
	if err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}

	var srv *http.Server
	apiLogger := logger.NewSystemLogger("api")
	serverErr := make(chan error, 1)

	if serverEnabled {
		srv = &http.Server{
			Addr:              cfg.Server.Address,
			Handler:           buildRouter(backupManager, sinks, version, uiEnabled),
			ReadHeaderTimeout: 10 * time.Second,
		}
		apiLogger.Info().Str("address", cfg.Server.Address).Bool("ui", uiEnabled).Msg("HTTP listening")

		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverErr <- err
			}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		systemLogger.Info().Str("signal", sig.String()).Msg("Shutting down")
	case err := <-serverErr:
		systemLogger.Error().Err(err).Msg("HTTP server failed")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	schedulerLogger := logger.NewSystemLogger("scheduler")
	if err := scheduler.Shutdown(); err != nil {
		schedulerLogger.Error().Err(err).Msg("Scheduler shutdown error")
	}

	if srv != nil {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			apiLogger.Error().Err(err).Msg("HTTP shutdown error")
		}
	}

	backupManager.Shutdown(shutdownCtx)
	systemLogger.Info().Msg("Stopped")
	return nil
}

func startScheduler(cfg *config.Config, manager *backup.Manager) (gocron.Scheduler, error) {
	lg := logger.NewSystemLogger("scheduler")
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	scheduled := 0
	for _, job := range cfg.Jobs {
		jobName := job.Name
		_, err := s.NewJob(
			gocron.CronJob(job.Schedule, false),
			gocron.NewTask(func() {
				lg.Debug().Str("job", jobName).Msg("Trigger")
				if err := manager.RunJob(jobName); err != nil {
					lg.Error().Err(err).Str("job", jobName).Msg("Job run failed")
				}
			}),
		)
		if err != nil {
			lg.Error().Err(err).
				Str("job", job.Name).
				Str("schedule", job.Schedule).
				Msg("Cannot schedule job")
			continue
		}
		scheduled++
		lg.Debug().Str("job", job.Name).Str("schedule", job.Schedule).Msg("Job scheduled")
	}
	lg.Info().Int("jobs", scheduled).Msg("Cron registered")

	s.Start()
	return s, nil
}

func buildRouter(manager *backup.Manager, sinks *logger.FileSinks, version string, uiEnabled bool) http.Handler {
	apiLogger := logger.NewSystemLogger("api")

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromXFFTrustedProxies(1))
	r.Use(middleware.Recoverer)
	r.Use(httpLogger(apiLogger))

	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	authMiddleware := api.NewAuthMiddleware(manager)
	api.MountAPI(r, manager, authMiddleware, sinks, version)

	if uiEnabled {
		r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("./web/dist/assets"))))
	}

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if !uiEnabled || strings.HasPrefix(req.URL.Path, "/api") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"Not found"}`))
			return
		}
		http.ServeFile(w, req, "./web/dist/index.html")
	})

	return r
}

func httpLogger(lg zerolog.Logger) func(http.Handler) http.Handler {
	skip := map[string]struct{}{"/metrics": {}}
	if utils.GetEnvironment() == "production" {
		skip["/api/v1/status"] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skip[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			clientIP := middleware.GetClientIP(r.Context())
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}

			lg.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Dur("latency", time.Since(start)).
				Str("client_ip", clientIP).
				Str("user_agent", r.UserAgent()).
				Msg("HTTP request")
		})
	}
}
