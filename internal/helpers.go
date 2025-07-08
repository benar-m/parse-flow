package internal

// Ring buffer to make use of frame if to avoid dupes incase of retries from logplex
func NewDedupeCache(s int) *DedupeCache {
	return &DedupeCache{
		Buffer:   make([]string, s),
		Lookup:   make(map[string]struct{}),
		Size:     s,
		WritePos: 0,
	}
}
func (d *DedupeCache) Add(msgId string) bool {
	if _, exists := d.Lookup[msgId]; exists {
		return false
	}

	//evict before insert
	e := d.Buffer[d.WritePos]
	if e != "" {
		delete(d.Lookup, e)
	}
	d.Buffer[d.WritePos] = msgId
	d.Lookup[msgId] = struct{}{}
	d.WritePos = (d.WritePos + 1) % d.Size

	return true

}
