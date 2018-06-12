package mgologstash

//Buffer represents an ipfix.Buffer backed by MongoDB and fed by Logstash
type Buffer interface {
	Next(out *Flow) bool
	Err() error
	Close()
}
