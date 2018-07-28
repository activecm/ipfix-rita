package buffered

import (
	"net"
	"time"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/output/rita"
	"github.com/activecm/ipfix-rita/converter/output/rita/buffered"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	rita_db "github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//bufferedRITAConnDateWriter writes session aggregates to MongoDB
//as RITA Conn records. Each record is routed
//to a database depending on the FlowEnd time. Additionally, it creates
//a RITA MetaDB record for each database before inserting data
//into the respective database. The data is batched up in buffers
//before being sent to MongoDB. The buffers are flushed when
//they are full or after a deadline passes for the individual buffer.
type bufferedRITAConnDateWriter struct {
	db                  rita.OutputDB
	localNets           []net.IPNet
	outputCollections   map[string]*buffered.AutoFlushCollection
	metaDBDatabasesColl *mgo.Collection
	bufferSize          int
	autoFlushTime       time.Duration
	log                 logging.Logger
}

//NewBufferedRITAConnDateWriter creates an buffered RITA compatible writer
//which splits records into different databases depending on the
//each record's flow end date. Metadatabase records are created
//as the output databases are created. Each buffer is flushed
//when the buffer is full or after a deadline passes.
func NewBufferedRITAConnDateWriter(ritaConf config.RITA, ipfixConf config.IPFIX, bufferSize int, autoFlushTime time.Duration, log logging.Logger) (output.SessionWriter, error) {
	db, err := rita.NewOutputDB(ritaConf)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to RITA MongoDB")
	}

	//parse local networks
	localNets, localNetsErrs := ipfixConf.GetLocalNetworks()
	if len(localNetsErrs) != 0 {
		for i := range localNetsErrs {
			log.Warn("could not parse local network", logging.Fields{"err": localNetsErrs[i]})
		}
	}
	//return the new writer
	return &bufferedRITAConnDateWriter{
		localNets:           localNets,
		outputCollections:   make(map[string]*buffered.AutoFlushCollection),
		metaDBDatabasesColl: db.NewMetaDBDatabasesConnection(),
		bufferSize:          bufferSize,
		autoFlushTime:       autoFlushTime,
		db:                  db,
		log:                 log,
	}, nil
}

func (r *bufferedRITAConnDateWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
	errs := make(chan error)
	go func() {
		defer close(errs)
		defer r.closeDBSessions()

		//loop over the input
		for sess := range sessions {
			//convert the record to RITA output
			var connRecord parsetypes.Conn
			sess.ToRITAConn(&connRecord, r.isIPLocal)

			//create/ get the buffered output collection
			outColl, ok := r.getConnCollectionForSession(sess, errs)
			if !ok {
				continue
			}

			//insert the record
			outColl.Insert(connRecord)
		}
	}()
	return errs
}

func (r *bufferedRITAConnDateWriter) closeDBSessions() {
	for i := range r.outputCollections {
		//stops outputCollections from sending on errs
		r.outputCollections[i].Close()
	}
	r.metaDBDatabasesColl.Database.Session.Close()
}

func (r *bufferedRITAConnDateWriter) isIPLocal(ipAddrStr string) bool {
	ipAddr := net.ParseIP(ipAddrStr)
	for i := range r.localNets {
		if r.localNets[i].Contains(ipAddr) {
			return true
		}
	}
	return false
}

func (r *bufferedRITAConnDateWriter) getConnCollectionForSession(sess *session.Aggregate, errs chan<- error) (*buffered.AutoFlushCollection, bool) {
	//get the latest flowEnd time
	endTimeMilliseconds := sess.FlowEndMillisecondsAB
	if sess.FlowEndMillisecondsBA > endTimeMilliseconds {
		endTimeMilliseconds = sess.FlowEndMillisecondsBA
	}
	//time.Unix(seconds, nanoseconds)
	//1000 milliseconds per second, 1000 nanosecodns to a microsecond. 1000 microseconds to a millisecond
	endTime := time.Unix(endTimeMilliseconds/1000, (endTimeMilliseconds%1000)*1000*1000)
	endTimeStr := endTime.Format("2006-01-02")

	//cache the database connection
	outBufferedColl, ok := r.outputCollections[endTimeStr]
	if !ok {
		//connect to the db
		var err error
		outColl, err := r.db.NewRITAOutputConnection(endTimeStr)
		if err != nil {
			errs <- errors.Wrapf(err, "could not connect to output database for suffix: %s", endTimeStr)
			return nil, false
		}

		//create the meta db record
		r.ensureMetaDBRecordExists(outColl.Database.Name)

		//create the output buffer
		outBufferedColl = buffered.NewAutoFlushCollection(outColl, r.bufferSize, r.autoFlushTime, errs)
		outBufferedColl.StartAutoFlush()

		//cache the result
		r.outputCollections[endTimeStr] = outBufferedColl
	}
	return outBufferedColl, true
}

func (r *bufferedRITAConnDateWriter) ensureMetaDBRecordExists(dbName string) error {
	numRecords, err := r.metaDBDatabasesColl.Find(bson.M{"name": dbName}).Count()
	if err != nil {
		return errors.Wrapf(err, "could not count MetaDB records with name: %s", dbName)
	}
	if numRecords != 0 {
		return nil
	}
	err = r.metaDBDatabasesColl.Insert(rita_db.DBMetaInfo{
		Name:           dbName,
		Analyzed:       false,
		ImportVersion:  "v1.0.0+ActiveCM-IPFIX",
		AnalyzeVersion: "",
	})
	if err != nil {
		return errors.Wrapf(err, "could not insert MetaDB record with name: %s", dbName)
	}
	return nil
}
