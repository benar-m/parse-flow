package main

import (
	"fmt"
	"log"
	"net/http"
	"parseflow/internal"

	ip2 "github.com/ip2location/ip2location-go"
)

func main() {
	dc := internal.NewDedupeCache(100)
	rawLogChan := make(chan []byte, 1000)
	parsedLogChan := make(chan *internal.ParsedLog, 100)
	db, err := ip2.OpenDB("./data/IP2LOCATION-LITE-DB1.IPV6.BIN")
	if err != nil {
		fmt.Println(err)
		log.Fatalf("Failed to open ip2location DB")
	}
	defer db.Close()

	app := &internal.App{
		Dc:            dc,
		RawLogChan:    rawLogChan,
		ParsedLogChan: parsedLogChan,
		GeoDb:         db,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /logdrains", app.LogReceiver)
	mux.HandleFunc("GET /metrics", app.MetricsHandler)
	// str := `2025-07-09T13:37:42.123456+00:00 heroku[router]: at=info method=GET path="/login" host=myapp.herokuapp.com request_id=123abc-456def fwd="197.248.10.42" dyno=web.1 connect=1ms service=23ms status=200 bytes=1345 protocol=https`
	// b := []byte(str)
	// fmt.Println(app.ParseLog(b))
	go func() {
		fmt.Println("Spy called")
		for log := range parsedLogChan {
			fmt.Println(log)
		}
	}()

	go func() {
		app.ParserWorker()
	}()
	app.StartMetricsAggregator()

	err = http.ListenAndServe(":5000", mux)
	if err != nil {
		log.Fatalf("Could not Start the Server: %v", err)
	}

}
