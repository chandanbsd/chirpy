package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	serveMux := http.NewServeMux()

	serveMux.Handle("/", http.FileServer(http.Dir(".")))

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
