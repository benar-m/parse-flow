package internal

import (
	"log"
	"strings"
)

func (a *App) ParseLog(logByte []byte) map[string]string {
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

	// Always send parsed logs (BuildParsedLog now returns valid struct with defaults)
	a.ParsedLogChan <- parsedlog

	return logParts

}

// Reads from RawLogChan and Parses
func (a *App) ParserWorker() {
	for logBytes := range a.RawLogChan {
		a.ParseLog(logBytes)
	}
}
