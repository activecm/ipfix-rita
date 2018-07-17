package output

import (
	"net"

	"github.com/activecm/ipfix-rita/converter/environment"
	"github.com/activecm/ipfix-rita/converter/stitching/session"
	rita_db "github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/pkg/errors"
)

//RITAConnWriter writes session aggregates to MongoDB
//as RITA Conn records. Additionally, it creates
//a RITA MetaDB record for the data once the program
//finishes executing.
type RITAConnWriter struct {
	environment.Environment
}

//Write converts session aggregates into RITA conn
//records and writes them out to MongoDB
//such that RITA can import the data
func (r RITAConnWriter) Write(sessions <-chan *session.Aggregate) <-chan error {
	errs := make(chan error)
	go func() {
		defer close(errs)

		outColl, err := r.DB.NewOutputConnection("")
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

		r.createMetaDBRecord(outColl.Database.Name)
	}()
	return errs
}

func (r RITAConnWriter) createMetaDBRecord(dbName string) {
	dbs := r.DB.NewMetaDBDatabasesConnection()
	dbs.Insert(rita_db.DBMetaInfo{
		Name:           dbName,
		Analyzed:       false,
		ImportVersion:  "v1.0.0+ActiveCM-IPFIX",
		AnalyzeVersion: "",
	})
}
