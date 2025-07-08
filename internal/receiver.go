package internal

import (
	"log"
	"net/http"
)

func LogReceiver(w http.ResponseWriter, r *http.Request) {
	//verify first
	if r.Header.Get("Content-Type") != "application/logplex-1" || r.Method != http.MethodPost {
		log.Println("Invalid Content")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

}
