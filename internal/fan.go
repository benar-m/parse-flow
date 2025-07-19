package internal

import "log"

func (a *App) FanOut() {
	for l := range a.ParsedLogChan {
		select {
		case a.MetricChan <- l:
		default:
			log.Printf("MetricChan full, dropping log from %s", l.SourceDyno)
		}
		select {
		case a.DbRawWriteChan <- l:
		default:
			log.Printf("DbRawWriteChan full, dropping log from %s", l.SourceDyno)
		}
	}
}
