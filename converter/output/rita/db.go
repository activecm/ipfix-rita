package rita

import (
	"github.com/activecm/ipfix-rita/converter/output/rita/buffered"
	"github.com/activecm/ipfix-rita/converter/output/rita/freqconn"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/globalsign/mgo"
	"github.com/pkg/errors"
	"time"
)

//DB represents a RITA database and provides routines
//for constructing a new RITA database and inserting
//new data into that database
type DB struct {
	manager         DBManager
	outputDB        *mgo.Database
	connColl        *buffered.AutoFlushCollection
	connCounter     freqconn.ConnCounter
	strobesNotifier freqconn.StrobesNotifier
}

func newDB(dbManager DBManager, outputDB *mgo.Database,
	strobeThreshold int, bufferSize int64, flushDeadline time.Duration,
	asyncErrorChan chan<- error, onFatalError func()) (DB, error) {

	strobesSess := outputDB.Session.Copy()
	connSess := outputDB.Session.Copy()

	strobesNotifier := freqconn.NewStrobesNotifier(outputDB.With(strobesSess))
	connCounter := freqconn.NewConnCounter(strobeThreshold, strobesNotifier)

	connColl := buffered.NewAutoFlushCollection(
		outputDB.C(RitaConnInputCollection).With(connSess),
		bufferSize, flushDeadline,
	)

	db := DB{
		manager:         dbManager,
		outputDB:        outputDB,
		connColl:        connColl,
		connCounter:     connCounter,
		strobesNotifier: strobesNotifier,
	}

	err := db.ensureConnIndexExists()
	if err != nil {
		strobesSess.Close()
		connSess.Close()
		return db, err
	}

	err = db.ensureFreqConnIndexExists()
	if err != nil {
		strobesSess.Close()
		connSess.Close()
		return db, err
	}

	started := connColl.StartAutoFlush(asyncErrorChan, onFatalError)
	if !started {
		err = errors.Errorf("failed to start auto flusher for collection %s.%s", outputDB.Name, RitaConnInputCollection)
		strobesSess.Close()
		connSess.Close()
		return db, err
	}

	err = dbManager.ensureMetaDBRecordExists(outputDB.Name)
	if err != nil {
		strobesSess.Close()
		connSess.Close()
		return db, err
	}

	return db, nil
}

func (d DB) ensureConnIndexExists() error {
	tmpConn := parsetypes.Conn{}
	for _, index := range tmpConn.Indices() {
		err := d.outputDB.C(RitaConnInputCollection).EnsureIndex(mgo.Index{
			Key: []string{index},
		})

		if err != nil {
			return errors.Wrapf(err, "could not create RITA conn index %s", index)
		}
	}
	return nil
}

func (d DB) ensureFreqConnIndexExists() error {
	tmpFreq := parsetypes.Freq{}
	for _, index := range tmpFreq.Indices() {
		err := d.outputDB.C(freqconn.StrobesCollection).EnsureIndex(mgo.Index{
			Key: []string{index},
		})

		if err != nil {
			return errors.Wrapf(err, "could not create RITA freqConn index %s", index)
		}
	}
	return nil
}

//InsertConnRecord writes a connection record to the RITA database.
//Each connection pair is counted, and if the count exceeds a threshold,
//the connection info is sent to freqConn, otherwise it is sent to conn
func (d DB) InsertConnRecord(connRecord *parsetypes.Conn) error {
	thresholdMet, err := d.connCounter.Increment(freqconn.UConnPair{
		Src: connRecord.Source,
		Dst: connRecord.Destination,
	})
	if err != nil {
		return err
	}
	if !thresholdMet {
		return d.connColl.Insert(connRecord)
	}
	return nil
}

//MarkFinished ensures that the database is ready for analysis
//by RITA. Note: MarkFinished may be called after Close. This is
//by design so the MetaDatabase is only updated after the last
//of the data has been flushed to MongoDB.
func (d DB) MarkFinished() error {
	return d.manager.markImportFinishedInMetaDB(d.outputDB.Name)
}

//Close closes the underlying database connections wrapped by the DB
func (d DB) Close() error {
	d.strobesNotifier.Close()
	//An error may arise when the collection is flushed in d.connColl.Close()
	err := d.connColl.Close()
	return err
}
