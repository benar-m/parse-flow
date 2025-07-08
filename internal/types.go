package internal

type App struct {
	Dc *DedupeCache
}
type DedupeCache struct {
	Buffer   []string //ring buffer
	Lookup   map[string]struct{}
	Size     int
	WritePos int //cuurent write position
}
