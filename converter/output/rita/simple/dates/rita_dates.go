package dates

import (
	"net"
	"time"

	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	rita_db "github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//simple.data.ritaConnDateWriter is not used in ipfix-rita but is
//kept around for convenience

//ritaConnDateWriter writes session aggregates to MongoDB
//as RITA Conn records. Each record is routed
//to a database depending on the FlowEnd time. Additionally, it creates
//a RITA MetaDB record for each database before inserting data
//into the respective database.
type ritaConnDateWriter struct {
	environment.Environment
	localNets           []net.IPNet
	outputCollections   map[string]*mgo.Collection
	metaDBDatabasesColl *mgo.Collection
}

//NewRITAConnDateWriter creates an unbuffered RITA compatible writer
//which splits records into different databases depending on the
//each record's flow end date. Metadatabase records are created
//as the output databases are created.
func NewRITAConnDateWriter(env environment.Environment) output.SessionWriter {
	localNets, localNetsErrs := env.GetIPFIXConfig().GetLocalNetworks()
	if len(localNetsErrs) != 0 {
		for i := range localNetsErrs {
			env.Logger.Warn("could not parse local network", logging.Fields{"err": localNetsErrs[i]})
		}
	}
	return &ritaConnDateWriter{
		Environment:         env,
		localNets:           localNets,
		outputCollections:   make(map[string]*mgo.Collection),
		metaDBDatabasesColl: env.DB.NewMetaDBDatabasesConnection(),
	}
}

func (r *ritaConnDateWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
	errs := make(chan error)
	go func() {
		defer close(errs)
		defer r.closeDBSessions()

		for sess := range sessions {
			var connRecord parsetypes.Conn
			sess.ToRITAConn(&connRecord, r.isIPLocal)
			outColl, err := r.getConnCollectionForSession(sess)
			if err != nil {
				errs <- errors.Wrapf(err, "could not connect to output collection for session:\n%+v", sess)
				continue
			}
			err = outColl.Insert(connRecord)
			if err != nil {
				errs <- errors.Wrapf(err, "could not insert conn record into output collection. conn record:\n%+v", connRecord)
				continue
			}
		}
	}()
	return errs
}

func (r *ritaConnDateWriter) closeDBSessions() {
	for _, coll := range r.outputCollections {
		coll.Database.Session.Close()
	}
	r.metaDBDatabasesColl.Database.Session.Close()
}

func (r *ritaConnDateWriter) isIPLocal(ipAddrStr string) bool {
	ipAddr := net.ParseIP(ipAddrStr)
	for i := range r.localNets {
		if r.localNets[i].Contains(ipAddr) {
			return true
		}
	}
	return false
}

func (r *ritaConnDateWriter) getConnCollectionForSession(sess *session.Aggregate) (*mgo.Collection, error) {
	endTimeMilliseconds := sess.FlowEndMillisecondsAB
	if sess.FlowEndMillisecondsBA > endTimeMilliseconds {
		endTimeMilliseconds = sess.FlowEndMillisecondsBA
	}
	//time.Unix(seconds, nanoseconds)
	//1000 milliseconds per second, 1000 nanosecodns to a microsecond. 1000 microseconds to a millisecond
	endTime := time.Unix(endTimeMilliseconds/1000, (endTimeMilliseconds%1000)*1000*1000)
	endTimeStr := endTime.Format("2006-01-02")

	//cache the database connection
	outColl, ok := r.outputCollections[endTimeStr]
	if !ok {
		var err error
		outColl, err = r.DB.NewRITAOutputConnection(endTimeStr)
		if err != nil {
			return nil, errors.Wrapf(err, "could not connect to output database for suffix: %s", endTimeStr)
		}
		r.ensureMetaDBRecordExists(outColl.Database.Name)
		r.outputCollections[endTimeStr] = outColl
	}
	return outColl, nil
}

func (r *ritaConnDateWriter) ensureMetaDBRecordExists(dbName string) error {
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
