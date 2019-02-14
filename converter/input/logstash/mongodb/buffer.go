package mongodb

import "github.com/activecm/ipfix-rita/converter/input/logstash/data"

//Buffer represents an iterator backed by MongoDB and fed by Logstash
type Buffer interface {
	Next(out *data.Flow) bool
	Err() error
	Close()
}
