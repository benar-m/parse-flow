package internal

//This is a manual broadcastor/fanner -- fans out
//reads from parsed log channel and then distributes this to the channels that need
// to remove competition from the parsed log channel

func (a *App) FanOut() {
	for l := range a.ParsedLogChan {
		a.MetricChan <- l
		a.DbRawWriteChan <- l

	}

}
