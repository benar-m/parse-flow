//Read from Parsed Log

//Brainstorm
/*
The idea is to have a central metrics channel
a routine comes across a 200, updates there
to avoid races, only one go routine will own that metrics channel, and the rest send data there and continue


resort to an actor model
A single go routines owns a metrics struct, and there exists a channel to wwhich
 parsers send data to (for now the function will be in metrics.go)
the go routine then updates its own struct
include a mutex lock since visualizer will need to read that struct to render a dashboard.

*/
