package main

import (
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	   return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg.fileserverHits.Add(1)
			next.ServeHTTP(w, r)
	   })
}

func healthzHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte("OK"))
}

func (cfg *apiConfig) hitsHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig)resetHandler(resWriter http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0);
}



func main() {
	serveMux := http.NewServeMux()

	var cfg = &apiConfig{
		fileserverHits: atomic.Int32{},
	}


	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	
	serveMux.HandleFunc("GET /admin/metrics", http.StripPrefix("/admin", http.FileServer(http.Dir("."))))
	serveMux.HandleFunc("GET /api/healthz", http.HandlerFunc(healthzHandler))

	serveMux.HandleFunc("GET /api/metrics", http.HandlerFunc(cfg.hitsHandler))

	serveMux.HandleFunc("POST /api/reset", http.HandlerFunc(cfg.resetHandler))

	server := http.Server {
		Handler: serveMux,
		Addr: ":8080",
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
