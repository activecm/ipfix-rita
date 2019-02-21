package rita

import (
	"time"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/rita/parser/parsetypes"
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

//RITADBManager wraps a *mgo.Session connected to MongoDB
//and provides facility for interacting with RITA compatible databases
type RITADBManager struct {
	ssn        *mgo.Session
	metaDBName string
	dbRoot     string
}

//NewRITADBManager instantiates a new RITAOutputDB with the
//details specified in the RITA configuration
func NewRITADBManager(ritaConf config.RITA) (RITADBManager, error) {
	db := RITADBManager{}
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

	db.ssn.DB(db.metaDBName).C(MetaDBDatabasesCollection).EnsureIndex(mgo.Index{
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
func (r RITADBManager) NewMetaDBDatabasesConnection() *mgo.Collection {
	return r.ssn.DB(r.metaDBName).C(MetaDBDatabasesCollection).With(r.ssn.Copy())
}

//NewRITAOutputConnection returns a new socket connected to the
//RITA output collection with a given DB suffix
func (r RITADBManager) NewRITAOutputConnection(dbNameSuffix string) (*mgo.Collection, error) {
	ssn := r.ssn.Copy()
	dbName := r.dbRoot
	if dbNameSuffix != "" {
		dbName = r.dbRoot + "-" + dbNameSuffix
	}

	//create the conn collection handle
	connColl := ssn.DB(dbName).C(RitaConnInputCollection)

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

	return connColl, nil
}

//EnsureMetaDBRecordExists ensures that a database record exists in the
//MetaDatabase for a given database name. This allows RITA to manage
//the database.
func (r RITADBManager) EnsureMetaDBRecordExists(dbName string) error {
	numRecords, err := r.ssn.DB(r.metaDBName).C(MetaDBDatabasesCollection).Find(bson.M{"name": dbName}).Count()
	if err != nil {
		return errors.Wrapf(err, "could not count MetaDB records with name: %s", dbName)
	}
	if numRecords != 0 {
		return nil
	}
	err = r.ssn.DB(r.metaDBName).C(MetaDBDatabasesCollection).Insert(DBMetaInfo{
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

//MarkImportFinishedInMetaDB sets the import_finished flag on the
//RITA MetaDatabase database record. This lets RITA know that no
//more data will be placed in the database and that the database
//is ready for analysis.
func (r RITADBManager) MarkImportFinishedInMetaDB(dbName string) error {
	err := r.ssn.DB(r.metaDBName).C(MetaDBDatabasesCollection).Update(
		bson.M{"name": dbName},
		bson.M{
			"$set": bson.M{
				"import_finished": true,
			},
		},
	)

	if err != nil {
		return errors.Wrapf(err, "could not mark database %s imported in database index %s.%s", dbName, r.metaDBName, MetaDBDatabasesCollection)
	}
	return nil
}

//Ping ensures the database connection is valid
func (r RITADBManager) Ping() error {
	err := r.ssn.Ping()
	if err != nil {
		return errors.Wrap(err, "could not contact the database")
	}
	//see if theres any permissions problems
	_, err = r.ssn.DatabaseNames()
	if err != nil {
		return errors.Wrap(err, "could not list the databases in the database")
	}
	return nil
}

//Close closing the underlying connection to MongoDB
func (r RITADBManager) Close() {
	r.ssn.Close()
}
