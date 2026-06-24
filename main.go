// Command funds runs the BIAN "Confirmation of Funds" service: the OBIE CBPII
// flow that confirms whether a debtor account holds a requested amount, against
// an authorised funds-confirmation consent.
package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sreeni/openbank-bian/pkg/accountscli"
	"github.com/sreeni/openbank-bian/pkg/consentcli"
	"github.com/sreeni/openbank-bian/pkg/httpx"
	"github.com/sreeni/openbank-bian/pkg/pg"
	"github.com/sreeni/openbank-bian/services/funds/internal/funds"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	addr := envOr("ADDR", ":8084")
	baseURL := envOr("BASE_URL", "http://localhost:8084")
	consentURL := envOr("CONSENT_URL", "http://localhost:8081")
	accountsURL := envOr("ACCOUNTS_URL", "http://localhost:8082")
	dsn := os.Getenv("DATABASE_URL")

	repo, err := newRepository(context.Background(), log, dsn)
	if err != nil {
		log.Error("init repository", "error", err)
		os.Exit(1)
	}

	consentClient := consentcli.New(consentURL)
	accountsClient := accountscli.New(accountsURL)
	svc := funds.NewService(repo, consentClient, accountsClient)
	handler := funds.NewHandler(svc, baseURL)

	root := httpx.Chain(handler.Routes(),
		httpx.FAPIInteractionID,
		httpx.Logger(log),
		httpx.Recoverer(log),
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("funds service listening",
			"addr", addr, "backend", backendName(dsn),
			"consent_url", consentURL, "accounts_url", accountsURL)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	shutdownOnSignal(log, srv)
}

// newRepository returns a Postgres repository when DATABASE_URL is set,
// otherwise an in-memory repository so the service runs with zero infra.
func newRepository(ctx context.Context, log *slog.Logger, dsn string) (funds.Repository, error) {
	if dsn == "" {
		log.Warn("DATABASE_URL not set, using in-memory store")
		return funds.NewMemRepository(), nil
	}
	pool, err := pg.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pg.RunMigrations(ctx, pool, migrations, "migrations", "funds"); err != nil {
		return nil, err
	}
	return funds.NewPgRepository(pool), nil
}

func shutdownOnSignal(log *slog.Logger, srv *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func backendName(dsn string) string {
	if dsn == "" {
		return "memory"
	}
	return "postgres"
}
