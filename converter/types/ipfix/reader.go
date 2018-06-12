package ipfix

import "context"

//Reader represents a source of IPFIX flows which can be read from
//asynchronously
type Reader interface {
	//Drain asynchronously drains a source of IPFIX flows
	Drain(context.Context) (<-chan Flow, <-chan error)
}
