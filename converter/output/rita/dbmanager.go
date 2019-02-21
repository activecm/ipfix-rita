package rita

import (
	"time"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/database"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

//MetaDBDatabasesCollection is the name of the RITA collection
//in the RITA MetaDB that keeps track of RITA managed databases
const MetaDBDatabasesCollection = "databases"

//RitaConnInputCollection is the name of the RITA collection
//which houses input connection data
const RitaConnInputCollection = "conn"

// Version specifies which RITA DB schema the resulting data matches
var Version = "v2.0.0+ActiveCM-IPFIX"

// TODO: Use version in RITA as dep

// DBMetaInfo defines some information about the database
type DBMetaInfo struct {
	ID             bson.ObjectId `bson:"_id,omitempty"`   // Ident
	Name           string        `bson:"name"`            // Name of the database
	ImportFinished bool          `bson:"import_finished"` // Has this database finished being imported
	Analyzed       bool          `bson:"analyzed"`        // Has this database been analyzed
	ImportVersion  string        `bson:"import_version"`  // Rita version at import
	AnalyzeVersion string        `bson:"analyze_version"` // Rita version at analyze
}

//DBManager wraps a *mgo.Session connected to MongoDB
//and provides facility for interacting with RITA compatible databases
type DBManager struct {
	ssn             *mgo.Session
	metaDBName      string
	dbRoot          string
	strobeThreshold int
	bufferSize      int64
	flushDeadline   time.Duration
}

//NewDBManager instantiates a new RITAOutputDB with the
//details specified in the RITA configuration
func NewDBManager(ritaConf config.RITA, strobeThreshold int,
	bufferSize int64, flushDeadline time.Duration) (DBManager, error) {

	db := DBManager{
		strobeThreshold: strobeThreshold,
		bufferSize:      bufferSize,
		flushDeadline:   flushDeadline,
	}

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

	err = db.ssn.DB(db.metaDBName).C(MetaDBDatabasesCollection).EnsureIndex(mgo.Index{
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
func (d DBManager) NewMetaDBDatabasesConnection() *mgo.Collection {
	return d.ssn.DB(d.metaDBName).C(MetaDBDatabasesCollection).With(d.ssn.Copy())
}

//NewRitaDB creates a new RITA Database by creating the appropriate
//MetaDB records, ensuring the correct indexes are in place, and returning
//a new rita.DB object. As data is written to the rita.DB object,
//data is continually flushed out to the database on another thread.
//If any errors occur on the flushing thread, they are reported on
//asyncErrorChan. If a fatal error occurs, onFatalError is called.
func (d DBManager) NewRitaDB(dbNameSuffix string, asyncErrorChan chan<- error, onFatalError func()) (DB, error) {
	dbName := d.dbRoot
	if dbNameSuffix != "" {
		dbName = d.dbRoot + "-" + dbNameSuffix
	}

	//note newDB will spawn off new sockets for connecting to MongoDB
	return newDB(
		d, d.ssn.DB(dbName),
		d.strobeThreshold, d.bufferSize, d.flushDeadline,
		asyncErrorChan, onFatalError,
	)
}

//ensureMetaDBRecordExists ensures that a database record exists in the
//MetaDatabase for a given database name. This allows RITA to manage
//the database.
func (d DBManager) ensureMetaDBRecordExists(dbName string) error {
	numRecords, err := d.ssn.DB(d.metaDBName).C(MetaDBDatabasesCollection).Find(bson.M{"name": dbName}).Count()
	if err != nil {
		return errors.Wrapf(err, "could not count MetaDB records with name: %s", dbName)
	}
	if numRecords != 0 {
		return nil
	}
	err = d.ssn.DB(d.metaDBName).C(MetaDBDatabasesCollection).Insert(DBMetaInfo{
		Name:           dbName,
		ImportFinished: false,
		Analyzed:       false,
		ImportVersion:  Version,
		AnalyzeVersion: "",
	})
	if err != nil {
		return errors.Wrapf(err, "could not insert MetaDB record with name: %s", dbName)
	}
	return nil
}

//markImportFinishedInMetaDB sets the import_finished flag on the
//RITA MetaDatabase database record. This lets RITA know that no
//more data will be placed in the database and that the database
//is ready for analysis.
func (d DBManager) markImportFinishedInMetaDB(dbName string) error {
	err := d.ssn.DB(d.metaDBName).C(MetaDBDatabasesCollection).Update(
		bson.M{"name": dbName},
		bson.M{
			"$set": bson.M{
				"import_finished": true,
			},
		},
	)

	if err != nil {
		return errors.Wrapf(err, "could not mark database %s imported in database index %s.%s", dbName, d.metaDBName, MetaDBDatabasesCollection)
	}
	return nil
}

//Ping ensures the database connection is valid
func (d DBManager) Ping() error {
	err := d.ssn.Ping()
	if err != nil {
		return errors.Wrap(err, "could not contact the database")
	}
	//see if theres any permissions problems
	_, err = d.ssn.DatabaseNames()
	if err != nil {
		return errors.Wrap(err, "could not list the databases in the database")
	}
	return nil
}

//Close closing the underlying connection to MongoDB
func (d DBManager) Close() {
	d.ssn.Close()
}
