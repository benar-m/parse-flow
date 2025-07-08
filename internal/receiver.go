package internal

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func (a *App) LogReceiver(w http.ResponseWriter, r *http.Request) {
	//verify first - specific to heroku -- (Parser will be compliant to RFC5424 on https drains)
	if r.Header.Get("Content-Type") != "application/logplex-1" || r.Method != http.MethodPost {
		log.Println("Invalid Content")
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !strings.HasPrefix(r.UserAgent(), "Logplex/v") {
		log.Println("Request Received From an unknown uA")
		w.Header().Set("Content-Lenght", "0")
		w.WriteHeader(http.StatusNoContent)
		return

	}
	msgLen := r.Header.Get("Logplex-Msg-Count")
	ml, err := strconv.Atoi(msgLen)
	if err != nil || ml < 1 {
		log.Println("Invalid message length")
		return
	}
	requestId := r.Header.Get("Logplex-Frame-Id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Internal Server Error: %v", err)
		return
	}
	defer r.Body.Close()

	if a.Dc.Add(requestId) {
		RawLogChan <- body

	} else {
		log.Println("Already Processed")
	}

}
