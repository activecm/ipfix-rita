package matching

import "github.com/activecm/ipfix-rita/converter/stitching/session"

//Matcher provides an interface for finding similar
//session.Aggregates based on the given session's
//AggregateQuery
type Matcher interface {
	//Close tears down any resources consumed by the Matcher
	//and flushes any remaining Aggregates from the matcher.
	Close() error
	//Find searches the Matcher for Aggregates which
	//match the given AggregateQuery. No other methods may be called
	//while Close() is in progress.
	Find(*session.AggregateQuery) session.Iterator
	//Insert adds a session aggregate to the Matcher.
	//Insert is responsible for setting the Aggregate.MatcherID field.
	//MatcherID must be used to disambiguate between aggregates in the
	//matcher with the same session.AggregateQuery. Usually MatcherID
	//is some sort of auto incrementing ID.
	Insert(*session.Aggregate) error
	//Update finds an Aggregate in the Matcher using the given
	//Aggregate's AggregateQuery and MatcherID and updates
	//the matching Aggregate's data.
	Update(*session.Aggregate) error
	//Remove finds an Aggregate in the Matcher using the given
	//Aggregate's AggregateQuery and MatcherID and removes it
	//from the system.
	Remove(*session.Aggregate) error
	//ShouldFlush returns true if Flush should be called in order
	//to maintain performance and ensure unmatched records are
	//written out in a timely manner.
	ShouldFlush() (bool, error)
	//Flush evicts Aggregates from the Matcher in order to maintain
	//performance and ensure unmatched records are written out in a
	//timely manner. No other methods may be called while Flush() is in progress.
	Flush() error
}
