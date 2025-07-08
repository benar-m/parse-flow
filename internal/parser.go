package internal

// A buffered RawLog channel will hold the Rawlogs (not parsed). Capacity 1k is arbitrary for now
var RawLogChan = make(chan []byte, 1000)

// A buffered ParsedLog channel will hold logs after parsing workers "parse" them
var ParsedLogChan = make(chan []byte, 1000)

//Define a Parseing functions

//Spawn workers
//write to ParsedLogs
