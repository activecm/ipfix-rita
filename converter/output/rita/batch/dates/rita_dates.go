package dates

import (
	"context"
	"net"
	"time"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/output/rita"
<<<<<<< HEAD
	"github.com/activecm/ipfix-rita/converter/output/rita/buffered"
=======
>>>>>>> f450d47... Update the batch dates writer
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/pkg/errors"
)

//batchRITAConnDateWriter writes session aggregates to MongoDB
//as RITA Conn records. Each record is routed
//to a database depending on the FlowEnd time. Additionally, it creates
//a RITA MetaDB record for each database before inserting data
//into the respective database. The data is batched up in buffers
//before being sent to MongoDB. The buffers are flushed when
//they are full or after a deadline passes for the individual buffer.
type batchRITAConnDateWriter struct {
<<<<<<< HEAD
	db                rita.DBManager
	localNets         []net.IPNet
	outputCollections map[string]*buffered.AutoFlushCollection
	bufferSize        int64
	autoFlushTime     time.Duration
	autoFlushContext  context.Context
	autoFlushOnFatal  func()
	log               logging.Logger
=======
	db               rita.DBManager
	localNets        []net.IPNet
	outputDBs        map[string]rita.DB
	autoFlushContext context.Context
	autoFlushOnFatal func()
	log              logging.Logger
>>>>>>> f450d47... Update the batch dates writer
}

//NewBatchRITAConnDateWriter creates an buffered RITA compatible writer
//which splits records into different databases depending on the
//each record's flow end date. Metadatabase records are created
//as the output databases are created. Each buffer is flushed
//when the buffer is full or after a deadline passes.
<<<<<<< HEAD
func NewBatchRITAConnDateWriter(ritaConf config.RITA, localNets []net.IPNet,
	bufferSize int64, autoFlushTime time.Duration, log logging.Logger) (output.SessionWriter, error) {
	db, err := rita.NewDBManager(ritaConf)
=======
func NewBatchRITAConnDateWriter(ritaConf config.RITA, ipfixConf config.IPFIX,
	bufferSize int64, autoFlushTime time.Duration, log logging.Logger) (output.SessionWriter, error) {
	db, err := rita.NewDBManager(ritaConf, bufferSize, autoFlushTime)
>>>>>>> f450d47... Update the batch dates writer
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to RITA MongoDB")
	}

<<<<<<< HEAD
	autoFlushContext, autoFlushOnFail := context.WithCancel(context.Background())
	//return the new writer
	return &batchRITAConnDateWriter{
		db:                db,
		localNets:         localNets,
		outputCollections: make(map[string]*buffered.AutoFlushCollection),
		bufferSize:        bufferSize,
		autoFlushTime:     autoFlushTime,
		autoFlushContext:  autoFlushContext,
		autoFlushOnFatal:  autoFlushOnFail,
		log:               log,
=======
	//parse local networks
	localNets, localNetsErrs := ipfixConf.GetLocalNetworks()
	if len(localNetsErrs) != 0 {
		for i := range localNetsErrs {
			log.Warn("could not parse local network", logging.Fields{"err": localNetsErrs[i]})
		}
	}

	autoFlushContext, autoFlushOnFail := context.WithCancel(context.Background())
	//return the new writer
	return &batchRITAConnDateWriter{
		db:               db,
		localNets:        localNets,
		outputDBs:        make(map[string]rita.DB),
		autoFlushContext: autoFlushContext,
		autoFlushOnFatal: autoFlushOnFail,
		log:              log,
>>>>>>> f450d47... Update the batch dates writer
	}, nil
}

func (r *batchRITAConnDateWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
	errs := make(chan error)
	go func() {
		defer close(errs)
		defer r.closeDBSessions(errs)

	WriteLoop:
		for {
			select {
			case <-r.autoFlushContext.Done():
				break WriteLoop
			case sess, ok := <-sessions:
				// check if the program is shutting down
				if !ok {
					break WriteLoop
				}
				// ensure there weren't any errors in the autoflusher
				// NOTE: select is nondeterministic, so sess may be selected
				// even though the context has triggered. This means we need
				// to check it again here.
				select {
				case <-r.autoFlushContext.Done():
					break WriteLoop
				default:
				}

				//convert the record to RITA output
				var connRecord parsetypes.Conn
				sess.ToRITAConn(&connRecord, r.isIPLocal)

				//create/ get the buffered output collection
<<<<<<< HEAD
				outColl, err := r.getConnCollectionForSession(sess, errs, r.autoFlushOnFatal)
=======
				outDB, err := r.getDBForSession(sess, errs, r.autoFlushOnFatal)
>>>>>>> f450d47... Update the batch dates writer
				if err != nil {
					errs <- err
					break WriteLoop
				}

				//insert the record
<<<<<<< HEAD
				err = outColl.Insert(connRecord)
=======
				err = outDB.InsertConnRecord(&connRecord)
>>>>>>> f450d47... Update the batch dates writer
				if err != nil {
					errs <- err
					break WriteLoop
				}
			}
		}
	}()
	return errs
}

func (r *batchRITAConnDateWriter) closeDBSessions(errs chan<- error) {
<<<<<<< HEAD
	for i := range r.outputCollections {
		r.outputCollections[i].Close()

		err := r.db.MarkImportFinishedInMetaDB(r.outputCollections[i].Database())
		//stops outputCollections from sending on errs
=======
	for i := range r.outputDBs {
		err := r.outputDBs[i].Close()
		if err != nil {
			errs <- err
		}

		err = r.outputDBs[i].MarkFinished()
>>>>>>> f450d47... Update the batch dates writer
		if err != nil {
			errs <- err
		}

	}
	r.db.Close()
}

func (r *batchRITAConnDateWriter) isIPLocal(ipAddrStr string) bool {
	ipAddr := net.ParseIP(ipAddrStr)
	for i := range r.localNets {
		if r.localNets[i].Contains(ipAddr) {
			return true
		}
	}
	return false
}

<<<<<<< HEAD
func (r *batchRITAConnDateWriter) getConnCollectionForSession(sess *session.Aggregate,
	autoFlushAsyncErrChan chan<- error, autoFlushOnFatal func()) (*buffered.AutoFlushCollection, error) {
=======
func (r *batchRITAConnDateWriter) getDBForSession(sess *session.Aggregate,
	autoFlushAsyncErrChan chan<- error, autoFlushOnFatal func()) (rita.DB, error) {
>>>>>>> f450d47... Update the batch dates writer

	//get the latest flowEnd time
	endTimeMilliseconds := sess.FlowEndMilliseconds()
	//time.Unix(seconds, nanoseconds)
	//1000 milliseconds per second, 1000 nanoseconds to a microsecond. 1000 microseconds to a millisecond
	endTime := time.Unix(endTimeMilliseconds/1000, (endTimeMilliseconds%1000)*1000*1000)
	endTimeStr := endTime.Format("2006-01-02")

	//cache the database connection
<<<<<<< HEAD
	outBufferedColl, ok := r.outputCollections[endTimeStr]
	if !ok {
		//connect to the db
		var err error
		outColl, err := r.db.NewRITAOutputConnection(endTimeStr)
		if err != nil {
			return nil, errors.Wrapf(err, "could not connect to output database for suffix: %s", endTimeStr)
		}

		//create the meta db record
		err = r.db.EnsureMetaDBRecordExists(outColl.Database.Name)
		if err != nil {
			outColl.Database.Session.Close()
			return nil, err
		}

		//create the output buffer
		outBufferedColl = buffered.NewAutoFlushCollection(outColl, r.bufferSize, r.autoFlushTime)
		outBufferedColl.StartAutoFlush(autoFlushAsyncErrChan, autoFlushOnFatal)

		//cache the result
		r.outputCollections[endTimeStr] = outBufferedColl
	}
	return outBufferedColl, nil
=======
	outDB, ok := r.outputDBs[endTimeStr]
	if !ok {
		//connect to the db
		var err error
		outDB, err := r.db.NewRitaDB(endTimeStr, autoFlushAsyncErrChan, autoFlushOnFatal)
		if err != nil {
			return outDB, errors.Wrapf(err, "could not create output database for suffix: %s", endTimeStr)
		}

		//cache the result
		r.outputDBs[endTimeStr] = outDB
	}
	return outDB, nil
>>>>>>> f450d47... Update the batch dates writer
}
