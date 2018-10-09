package rita

import (
	"time"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/database"
	rita_db "github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/parsetypes"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

//metaDBDatabasesCollection is the name of the RITA collection
//in the RITA MetaDB that keeps track of RITA managed databases
const metaDBDatabasesCollection = "databases"

//ritaConnInputCollection is the name of the RITA collection
//which houses input connection data
const ritaConnInputCollection = "conn"

//OutputDB wraps a *mgo.Session connected to MongoDB
//and provides facility for interacting with RITA compatible databases
type OutputDB struct {
	ssn        *mgo.Session
	metaDBName string
	dbRoot     string
}

//NewOutputDB instantiates a new RITAOutputDB with the
//details specified in the RITA configuration
func NewOutputDB(ritaConf config.RITA) (OutputDB, error) {
	db := OutputDB{}
	var err error
	db.ssn, err = database.Dial(ritaConf.GetConnectionConfig())
	if err != nil {
		return db, err
	}
	db.ssn.SetSocketTimeout(1 * time.Hour)
	db.ssn.SetSyncTimeout(1 * time.Hour)
	db.ssn.SetCursorTimeout(0)

	db.dbRoot = ritaConf.GetDBRoot()
	db.metaDBName = ritaConf.GetMetaDB()

	db.ssn.DB(db.metaDBName).C(metaDBDatabasesCollection).EnsureIndex(mgo.Index{
		Key: []string{
			"name",
		},
		Unique:   true,
		DropDups: true,
		Name:     "nameindex",
	})

	if err != nil {
		db.ssn.Close()
		return db, errors.Wrap(err, "could not create MetaDB nameindex index")
	}

	return db, nil
}

//NewMetaDBDatabasesConnection returns a new socket connected to the
//MetaDB databases collection
func (o OutputDB) NewMetaDBDatabasesConnection() *mgo.Collection {
	return o.ssn.DB(o.metaDBName).C(metaDBDatabasesCollection).With(o.ssn.Copy())
}

//NewRITAOutputConnection returns a new socket connected to the
//RITA output collection with a given DB suffix
func (o OutputDB) NewRITAOutputConnection(dbNameSuffix string) (*mgo.Collection, error) {
	ssn := o.ssn.Copy()
	dbName := o.dbRoot
	if dbNameSuffix != "" {
		dbName = o.dbRoot + "-" + dbNameSuffix
	}

	//create the conn collection handle
	connColl := ssn.DB(dbName).C(ritaConnInputCollection)

	//ensure RITA's needed indexes exist
	tmpConn := parsetypes.Conn{}
	for _, index := range tmpConn.Indices() {
		err := connColl.EnsureIndex(mgo.Index{
			Key: []string{index},
		})

		if err != nil {
			ssn.Close()
			return nil, errors.Wrapf(err, "could not create RITA conn index %s", index)
		}
	}

	//ensure a MetaDB record exists for the collection
	err := o.ensureMetaDBRecordExists(dbName)
	if err != nil {
		ssn.Close()
		return nil, err //no need to wrap, wrapped in method
	}

	return connColl, nil
}

func (o OutputDB) ensureMetaDBRecordExists(dbName string) error {
	numRecords, err := o.ssn.DB(o.metaDBName).C(metaDBDatabasesCollection).Find(bson.M{"name": dbName}).Count()
	if err != nil {
		return errors.Wrapf(err, "could not count MetaDB records with name: %s", dbName)
	}
	if numRecords != 0 {
		return nil
	}
	err = o.ssn.DB(o.metaDBName).C(metaDBDatabasesCollection).Insert(rita_db.DBMetaInfo{
		Name:           dbName,
		Analyzed:       false,
		ImportVersion:  "v1.0.3+ActiveCM-IPFIX",
		AnalyzeVersion: "",
	})
	if err != nil {
		return errors.Wrapf(err, "could not insert MetaDB record with name: %s", dbName)
	}
	return nil
}

//Ping ensures the database connection is valid
func (o OutputDB) Ping() error {
	err := o.ssn.Ping()
	if err != nil {
		return errors.Wrap(err, "could not contact the database")
	}
	//see if theres any permissions problems
	_, err = o.ssn.DatabaseNames()
	if err != nil {
		return errors.Wrap(err, "could not list the databases in the database")
	}
	return nil
}

//Close closing the underlying connection to MongoDB
func (o OutputDB) Close() {
	o.ssn.Close()
}
