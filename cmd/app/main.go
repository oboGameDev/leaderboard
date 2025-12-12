package app

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appinternal "github.com/oboGameDev/leaderboard/internal/applogic"
	cfgpkg "github.com/oboGameDev/leaderboard/internal/config"
	httpapi "github.com/oboGameDev/leaderboard/internal/httpserver"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config yaml")
	flag.Parse()

	cfg, err := cfgpkg.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	app, err := appinternal.NewAppFromConfig(cfg)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	handler := httpapi.NewHandler(app)

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	go func() {
		log.Printf("http listening %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http serve: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Print("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
	log.Print("done")
}
