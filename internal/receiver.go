package internal

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// remove the syslog priority and version from Heroku log
func stripSyslogPrefix(logLine string) string {
	if len(logLine) == 0 || logLine[0] != '<' {
		return logLine
	}

	// Find closing
	closeIdx := strings.Index(logLine, ">")
	if closeIdx == -1 {
		return logLine
	}
	remaining := logLine[closeIdx+1:]
	spaceIdx := strings.Index(remaining, " ")
	if spaceIdx == -1 {
		return logLine
	}

	return remaining[spaceIdx+1:]
}

// Handles a post request from the logplexer and verifies then writes the log to Raw Log Chan
func (a *App) LogReceiver(w http.ResponseWriter, r *http.Request) {
	//verify first - specific to heroku -- (Parser will be compliant to RFC5424 on https drains)
	defer r.Body.Close()
	if r.Header.Get("Content-Type") != "application/logplex-1" || r.Method != http.MethodPost {
		log.Println("Invalid Content")
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	userAgent := r.UserAgent()
	if !strings.HasPrefix(userAgent, "Logplex/v") && !strings.HasPrefix(userAgent, "logfwd") {
		log.Printf("Request Received From %v\n", userAgent)
		w.Header().Set("Content-Length", "0")
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

	if a.Dc.Add(requestId) {
		cleanedBody := stripSyslogPrefix(string(body))

		//Filter to Router logs for now, will handle system logs later
		if strings.Contains(cleanedBody, "heroku router") {
			a.RawLogChan <- []byte(cleanedBody)
		} else {
			log.Printf("Skipping non-router log: %s", cleanedBody)
		}
	} else {
		log.Println("Already Processed")
	}

}
