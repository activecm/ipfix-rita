package stitching

import (
	"context"
	"math"

	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type flusher struct {
	exporterAddress string
}

func newFlusher(exporterAddress string) flusher {
	return flusher{
		exporterAddress: exporterAddress,
	}
}

func (f flusher) appendMaxExpireTime(stitcherID int, maxExpireTime uint64) {
	//TODO: Push into structure
}

func (f flusher) stitcherDone(stitcherID int) {
	//TODO: Pop structure
}

func (f flusher) run(ctx context.Context, doneSignal chan<- struct{},
	sessionsColl *mgo.Collection, writer output.SessionWriter) {
	for {
		select {
		case <-ctx.Done():
			err := f.flushSession(uint64(math.MaxInt64), sessionsColl, writer)
			for err == nil {
				err = f.flushSession(uint64(math.MaxInt64), sessionsColl, writer)
			}
			close(doneSignal)
			return
			/*
				default:
					TODO: pull minMaxExpireTime from structure
					minMaxExpireTime := uint64(0)
					f.flushSession(minMaxExpireTime, sessionsColl, writer)
			*/
		}
	}
}

func (f flusher) flushSession(maxExpireTime uint64, sessionsColl *mgo.Collection, writer output.SessionWriter) error {
	var oldSession session.Aggregate
	_, err := sessionsColl.Find(bson.M{
		"flowEndMillisecondsAB": bson.M{
			"$lt": maxExpireTime,
		},
		"flowEndMillisecondsBA": bson.M{
			"$lt": maxExpireTime,
		},
		"exporter": f.exporterAddress,
	}).Apply(mgo.Change{
		Remove: true,
	}, &oldSession)

	if err != nil {
		return err
	}

	return writer.Write(&oldSession)
}
