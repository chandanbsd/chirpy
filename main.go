package main

import (
	"encoding/json"
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
	return
}

func (cfg *apiConfig) hitsHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) resetHandler(resWriter http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0);
}

func (cfg *apiConfig) metricsHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Context-Type", "text/html")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte(fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handleValidateChirp(resWriter http.ResponseWriter, req *http.Request) {

	type payload struct {
		body string `json:"body"`
	}

	type dto struct {
		error string `json:"error"`
	}

	type dtoValid struct {
		body string `json:"body"`
	}

	defer req.Body.Close()
	decode := json.NewDecoder(req.Body)


	var p payload

	err := decode.Decode(&p)
	if err != nil {
		d := dtoValid {
			body:  "This is an opinion I need to share with the world",
		}	
		dataBytes, _ := json.Marshal(d)

		resWriter.WriteHeader(400)
		resWriter.Write(dataBytes)
	} else {
		d := dto {
			error: "Something went wrong",
		}	
		dataBytes, _ := json.Marshal(d)
		resWriter.WriteHeader(200)
		resWriter.Write(dataBytes)
	}
}


func main() {
	serveMux := http.NewServeMux()

	var cfg = &apiConfig{
		fileserverHits: atomic.Int32{},
	}

	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	
	serveMux.Handle("/admin/metrics/", http.HandlerFunc(cfg.metricsHandler))
	serveMux.Handle("GET /api/healthz", http.HandlerFunc(healthzHandler))

	serveMux.Handle("GET /api/metrics", http.HandlerFunc(cfg.hitsHandler))

	serveMux.Handle("POST /admin/reset", http.HandlerFunc(cfg.resetHandler))
	serveMux.Handle("POST /api/validate_chirp", http.HandlerFunc(cfg.handleValidateChirp))

	server := http.Server {
		Handler: serveMux,
		Addr:    ":8080",
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
