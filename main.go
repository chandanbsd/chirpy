package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/chandanbsd/chirpy/handler"
	"github.com/chandanbsd/chirpy/internal/database"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Failed to load .env filgo dot env: %v\n", err)
		os.Exit(1)
	}
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)

	serveMux := http.NewServeMux()

	var cfg = &handler.ApiConfig{
		FileserverHits: atomic.Int32{},
		Queries:        database.New(db),
	}

	serveMux.Handle("/app/", cfg.MiddlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	serveMux.Handle("/admin/metrics/", http.HandlerFunc(cfg.MetricsHandler))
	serveMux.Handle("GET /api/healthz", http.HandlerFunc(handler.HealthzHandler))

	serveMux.Handle("GET /api/metrics", http.HandlerFunc(cfg.HitsHandler))

	serveMux.Handle("POST /admin/reset", http.HandlerFunc(cfg.ResetHandler))
	serveMux.Handle("POST /api/validate_chirp", http.HandlerFunc(cfg.HandleValidateChirp))

	server := http.Server{
		Handler: serveMux,
		Addr:    ":8080",
	}

	err = server.ListenAndServe()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
