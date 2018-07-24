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

type RITAConnDateWriter struct {
	environment.Environment
	localNets           []net.IPNet
	outputCollections   map[string]*mgo.Collection
	metaDBDatabasesColl *mgo.Collection
}

func NewRITAConnDateWriter(env environment.Environment) output.SessionWriter {
	localNets, localNetsErrs := env.GetIPFIXConfig().GetLocalNetworks()
	if len(localNetsErrs) != 0 {
		for i := range localNetsErrs {
			env.Logger.Warn("could not parse local network", logging.Fields{"err": localNetsErrs[i]})
		}
	}
	return &RITAConnDateWriter{
		Environment:         env,
		localNets:           localNets,
		outputCollections:   make(map[string]*mgo.Collection),
		metaDBDatabasesColl: env.DB.NewMetaDBDatabasesConnection(),
	}
}

func (r *RITAConnDateWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
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

func (r *RITAConnDateWriter) closeDBSessions() {
	for _, coll := range r.outputCollections {
		coll.Database.Session.Close()
	}
	r.metaDBDatabasesColl.Database.Session.Close()
}

func (r *RITAConnDateWriter) isIPLocal(ipAddrStr string) bool {
	ipAddr := net.ParseIP(ipAddrStr)
	for i := range r.localNets {
		if r.localNets[i].Contains(ipAddr) {
			return true
		}
	}
	return false
}

func (r *RITAConnDateWriter) getConnCollectionForSession(sess *session.Aggregate) (*mgo.Collection, error) {
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
		outColl, err = r.DB.NewOutputConnection(endTimeStr)
		if err != nil {
			return nil, errors.Wrapf(err, "could not connect to output database for suffix: %s", endTimeStr)
		}
		r.ensureMetaDBRecordExists(outColl.Database.Name)
		r.outputCollections[endTimeStr] = outColl
	}
	return outColl, nil
}

func (r *RITAConnDateWriter) ensureMetaDBRecordExists(dbName string) error {
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
