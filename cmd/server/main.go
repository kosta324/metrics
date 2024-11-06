package main

import (
	"fmt"
	"net/http"

	"github.com/kosta324/metrics.git/internal/handlers"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	repo := handlers.InitStorage()
	handler := handlers.NewHandler{
		Repo: &repo,
	}
	mux := http.NewServeMux()
	handler.Handle(mux)
	fmt.Printf("Server run")
	return http.ListenAndServe(`:8080`, mux)
}
