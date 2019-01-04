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

//TODO: Use version in RITA as dep
// DBMetaInfo defines some information about the database
type DBMetaInfo struct {
	ID             bson.ObjectId `bson:"_id,omitempty"`   // Ident
	Name           string        `bson:"name"`            // Name of the database
	ImportFinished bool          `bson:"import_finished"` // Has this database finished being imported
	Analyzed       bool          `bson:"analyzed"`        // Has this database been analyzed
	ImportVersion  string        `bson:"import_version"`  // Rita version at import
	AnalyzeVersion string        `bson:"analyze_version"` // Rita version at analyze
}

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
func (o OutputDB) NewMetaDBDatabasesConnection() *mgo.Collection {
	return o.ssn.DB(o.metaDBName).C(MetaDBDatabasesCollection).With(o.ssn.Copy())
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
func (o OutputDB) EnsureMetaDBRecordExists(dbName string) error {
	numRecords, err := o.ssn.DB(o.metaDBName).C(MetaDBDatabasesCollection).Find(bson.M{"name": dbName}).Count()
	if err != nil {
		return errors.Wrapf(err, "could not count MetaDB records with name: %s", dbName)
	}
	if numRecords != 0 {
		return nil
	}
	err = o.ssn.DB(o.metaDBName).C(MetaDBDatabasesCollection).Insert(DBMetaInfo{
		Name:           dbName,
		ImportFinished: false,
		Analyzed:       false,
		ImportVersion:  "v2.0.0+ActiveCM-IPFIX",
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
func (o OutputDB) MarkImportFinishedInMetaDB(dbName string) error {
	err := o.ssn.DB(o.metaDBName).C(MetaDBDatabasesCollection).Update(
		bson.M{"name": dbName},
		bson.M{
			"$set": bson.M{
				"import_finished": true,
			},
		},
	)

	if err != nil {
		return errors.Wrapf(err, "could not mark database %s imported in database index %s.%s", dbName, o.metaDBName, MetaDBDatabasesCollection)
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
