package simple

import (
	"net"

	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/output"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	rita_db "github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

//ritaConnWriter is not used in ipfix-rita as it is too slow

//ritaConnWriter writes session aggregates to MongoDB
//as RITA Conn records. Additionally, it creates
//a RITA MetaDB record for the data once the program
//finishes executing. Only a single output database is used.
type ritaConnWriter struct {
	environment.Environment
}

//NewRITAConnWriter creates a new ritaConnWriter which
//writes session aggregate records to MongoDB in a RITA compatible format
//to a single database. The MetaDB entry is created as the writer exits.
func NewRITAConnWriter(env environment.Environment) output.SessionWriter {
	return ritaConnWriter{
		Environment: env,
	}
}

//Write converts session aggregates into RITA conn
//records and writes them out to MongoDB
//such that RITA can import the data
func (r ritaConnWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
	errs := make(chan error)
	go func() {
		defer close(errs)

		outColl, err := r.DB.NewRITAOutputConnection("")
		if err != nil {
			errs <- errors.Wrap(err, "could not connect to output collection")
			return
		}

		localNets, localNetsErrs := r.GetIPFIXConfig().GetLocalNetworks()
		if len(localNetsErrs) != 0 {
			for i := range localNetsErrs {
				errs <- errors.Wrap(localNetsErrs[i], "could not parse local network")
			}
		}

		for sess := range sessions {
			var connRecord parsetypes.Conn

			sess.ToRITAConn(&connRecord, func(ipAddrStr string) bool {
				ipAddr := net.ParseIP(ipAddrStr)
				for i := range localNets {
					if localNets[i].Contains(ipAddr) {
						return true
					}
				}
				return false
			})

			err := outColl.Insert(connRecord)
			if err != nil {
				errs <- errors.Wrapf(err, "could not insert record into RITA conn collection\n%+v", connRecord)
			}
		}

		r.ensureMetaDBRecordExists(outColl.Database.Name)
	}()
	return errs
}

func (r *ritaConnWriter) ensureMetaDBRecordExists(dbName string) error {
	dbs := r.DB.NewMetaDBDatabasesConnection()
	defer dbs.Database.Session.Close()

	numRecords, err := dbs.Find(bson.M{"name": dbName}).Count()
	if err != nil {
		return errors.Wrapf(err, "could not count MetaDB records with name: %s", dbName)
	}
	if numRecords != 0 {
		return nil
	}
	err = dbs.Insert(rita_db.DBMetaInfo{
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
