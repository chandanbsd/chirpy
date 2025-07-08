package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	serveMux := http.NewServeMux()

	server := http.Server {
		Handler: serveMux,
		Addr: ":8080",
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Failed to start server")
		os.Exit(1)
	}
}
