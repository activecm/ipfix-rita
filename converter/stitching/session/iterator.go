package session

//Iterator provides an interface for iterating
//over a collection of session.Aggregates
type Iterator interface {
	Next(*Aggregate) bool
	Err() error
}
