package mgologstash

import mgo "gopkg.in/mgo.v2"

//idBuffer works by selecting and removing the least recently inserted
//record in an input collection
type idBuffer struct {
	input *mgo.Collection
	err   error
}

//NewIDBuffer returns an ipfix.Buffer backed by MongoDB and fed by Logstash
func NewIDBuffer(input *mgo.Collection) Buffer {
	return &idBuffer{
		input: input,
	}
}

//Next returns the next record that was inserted into the input collection.
//Next returns false if there is no more data. Next may set an error when
//it returns false. This error can be read with Err()
func (b *idBuffer) Next(out *Flow) bool {
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

//Err returns any errors set by Read()
func (b *idBuffer) Err() error {
	return b.err
}

//Close closes the socket to the MongoDB server
func (b *idBuffer) Close() {
	b.input.Database.Session.Close()
}