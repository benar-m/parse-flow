package main

import (
	"log"
	"net/http"
	"parseflow/internal"
)

func main() {
	dc := internal.NewDedupeCache(100)

	app := &internal.App{
		Dc: dc,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /logdrains", app.LogReceiver)

	err := http.ListenAndServe(":5000", mux)
	if err != nil {
		log.Fatalf("Could not Start the Server: %v", err)
	}

}
