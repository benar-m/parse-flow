package internal

import (
	"log"
	"strings"
)

//Define a Parseing functions

//Spawn workers
//write to ParsedLogs

func (a *App) ParseLog(logByte []byte) map[string]string {
	//
	//2025-07-09T13:42:00.789123+00:00 heroku[router]: at=error method=POST path="/api/data" host=myapp.herokuapp.com request_id=xyz-789abc fwd="102.130.55.77" dyno=web.1 connect=0ms service=503ms status=500 bytes=512 protocol=https

	logString := string(logByte)
	fields := strings.Fields(logString)
	logParts := make(map[string]string)

	for _, f := range fields {
		if strings.Contains(f, "=") {
			parts := strings.SplitN(f, "=", 2)
			k := parts[0]
			v := parts[1]
			logParts[k] = v

		}
	}
	f := strings.SplitN(logString, " ", 2)
	if len(f) < 2 {
		log.Println("Malformed Request Received")
		return map[string]string{}
	}
	timeStampstr := f[0]
	logParts["timestamp"] = timeStampstr
	parsedlog := BuildParsedLog(logParts)
	a.ParsedLogChan <- parsedlog

	return logParts

}
