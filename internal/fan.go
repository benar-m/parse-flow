package internal

import "log"

func (a *App) FanOut() {
	for l := range a.ParsedLogChan {
		// Block on MetricChan - metrics are critical
		a.MetricChan <- l

		// Non-blocking send to DB with proper logging
		select {
		case a.DbRawWriteChan <- l:
		default:
			log.Printf("WARNING: DbRawWriteChan full, log data may be lost from %s", l.SourceDyno)
		}
	}
}
