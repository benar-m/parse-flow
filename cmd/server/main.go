package main

import (
	"log"
	"net/http"
	"parseflow/internal"
)

func main() {
	dc := internal.NewDedupeCache(100)
	rawLogChan := make(chan []byte, 1000)
	parsedLogChan := make(chan *internal.ParsedLog, 100)

	app := &internal.App{
		Dc:            dc,
		RawLogChan:    rawLogChan,
		ParsedLogChan: parsedLogChan,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /logdrains", app.LogReceiver)
	// str := `2025-07-09T13:37:42.123456+00:00 heroku[router]: at=info method=GET path="/login" host=myapp.herokuapp.com request_id=123abc-456def fwd="197.248.10.42" dyno=web.1 connect=1ms service=23ms status=200 bytes=1345 protocol=https`
	// b := []byte(str)
	// fmt.Println(app.ParseLog(b))

	err := http.ListenAndServe(":5000", mux)
	if err != nil {
		log.Fatalf("Could not Start the Server: %v", err)
	}

}
