package mgologstash

import (
	mgo "gopkg.in/mgo.v2"
)

//Buffer represents an ipfix.Buffer backed by MongoDB and fed by Logstash
type Buffer struct {
	input *mgo.Collection
	err   error
}

//NewBuffer returns an ipfix.Buffer backed by MongoDB and fed by Logstash
func NewBuffer(input *mgo.Collection) *Buffer {
	return &Buffer{
		input: input,
	}
}

func (b *Buffer) Next(out *Flow) bool {
	_, err := b.input.Find(nil).Sort("_id").Apply(
		mgo.Change{
			Remove: true,
		},
		&out,
	)
	if err != nil {
		if err != mgo.ErrNotFound {
			b.err = err
		}
		return false
	}

	return true
}

func (b *Buffer) Err() error {
	return b.err
}

func (b *Buffer) Close() error {
	b.input.Database.Session.Close()
	return nil
}
