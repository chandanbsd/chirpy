package main

import (
	"fmt"
	"net/http"
	"os"
)

func healthzHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte("OK"))
}

func main() {
	serveMux := http.NewServeMux()

	serveMux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))

	serveMux.HandleFunc("/healthz", healthzHandler)

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
