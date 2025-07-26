package main

import (
	"fmt"
	"log"
	"net/http"
	"parseflow/internal"

	ip2 "github.com/ip2location/ip2location-go"
)

func main() {
	config := internal.LoadConfig()

	dc := internal.NewDedupeCache(100)
	rawLogChan := make(chan []byte, config.RawLogChanSize)
	parsedLogChan := make(chan *internal.ParsedLog, config.ParsedLogChanSize)
	db, err := ip2.OpenDB("./data/IP2LOCATION-LITE-DB1.IPV6.BIN")
	if err != nil {
		fmt.Println(err)
		log.Fatalf("Failed to open ip2location DB")
	}
	defer db.Close()
	rawDbChan := make(chan *internal.ParsedLog, config.RawLogChanSize)
	dbWriteChan := make(chan *internal.Metric, 1000) //metric should <- to this on processing flow
	metricChan := make(chan *internal.ParsedLog, config.MetricChanSize)

	app := &internal.App{
		Dc:             dc,
		RawLogChan:     rawLogChan,
		ParsedLogChan:  parsedLogChan,
		GeoDb:          db,
		DbRawWriteChan: rawDbChan,
		DbWriteChan:    dbWriteChan,
		MetricChan:     metricChan,
		Config:         config,
	}
	app.RateLimiter = internal.NewRateLimiterMap(100, 10)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /logdrains", app.LogReceiver)
	mux.HandleFunc("GET /metrics", app.MetricsHandler)

	go app.ParserWorker()
	go app.FanOut()
	go app.StartMetricsAggregator()
	go app.StartDbWriter()

	err = http.ListenAndServe(":"+config.Port, mux)
	if err != nil {
		log.Fatalf("Could not Start the Server: %v", err)
	}

}
