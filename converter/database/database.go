package database

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/mgosec"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
)

//metaDBDatabasesCollection is the name of the RITA collection
//in the RITA MetaDB that keeps track of RITA managed databases
const metaDBDatabasesCollection = "databases"

//ritaConnInputCollection is the name of the RITA collection
//which houses input connection data
const ritaConnInputCollection = "conn"

//DB extends mgo.Session with application specific functionality
type DB struct {
	ssn             *mgo.Session
	inputDB         string
	inputColl       *mgo.Collection
	sessions        *mgo.Collection
	metaDBDatabases *mgo.Collection
	outputDBRoot    string
}

//NewDB creates a mongodb session based on the mongo config
func NewDB(mongoConfiguration config.MongoDB, ritaConfiguration config.RITA) (DB, error) {
	db := DB{}
	var err error
	if mongoConfiguration.GetTLS().IsEnabled() {
		db.ssn, err = dialTLS(mongoConfiguration)
		err = errors.Wrap(err, "could not connect to MongoDB over TLS")
	} else {
		db.ssn, err = dialInsecure(mongoConfiguration)
		err = errors.Wrap(err, "could not connect to MongoDB (no TLS)")
	}
	if err != nil {
		return db, err
	}
	db.ssn.SetSocketTimeout(mongoConfiguration.GetSocketTimeout())
	db.ssn.SetSyncTimeout(mongoConfiguration.GetSocketTimeout())
	db.ssn.SetCursorTimeout(0)

	db.inputDB = mongoConfiguration.GetDatabase()
	db.inputColl = db.ssn.DB(db.inputDB).C(mongoConfiguration.GetCollection())
	db.outputDBRoot = ritaConfiguration.GetDBRoot()

	//ensure the meta database is set up
	db.metaDBDatabases = db.ssn.DB(ritaConfiguration.GetMetaDB()).C(metaDBDatabasesCollection)
	err = db.metaDBDatabases.EnsureIndex(mgo.Index{
		Key: []string{
			"name",
		},
		Unique:   true,
		DropDups: true,
		Name:     "nameindex",
	})

	if err != nil {
		return db, errors.Wrap(err, "could not create MetaDB nameindex index")
	}

	return db, nil
}

//NewHelperCollection returns a new socket connected
//to a MongoDB collection in the input database (mongoConfiguration.GetDatabase())
//with the given name collName.
func (db *DB) NewHelperCollection(collName string) *mgo.Collection {
	return db.ssn.Copy().DB(db.inputDB).C(collName)
}

//NewInputConnection returns a new socket connected to the input
//collection
func (db *DB) NewInputConnection() *mgo.Collection {
	ssn := db.ssn.Copy()
	return db.inputColl.With(ssn)
}

//NewMetaDBDatabasesConnection returns a new socket connected to the
//MetaDB databases collection
func (db *DB) NewMetaDBDatabasesConnection() *mgo.Collection {
	ssn := db.ssn.Copy()
	return db.metaDBDatabases.With(ssn)
}

//NewRITAOutputConnection returns a new socket connected to the
//RITA output collection with a given DB suffix
func (db *DB) NewRITAOutputConnection(dbNameSuffix string) (*mgo.Collection, error) {
	ssn := db.ssn.Copy()
	dbName := db.outputDBRoot
	if dbNameSuffix != "" {
		dbName = db.outputDBRoot + "-" + dbNameSuffix
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
	return connColl, nil
}

//Ping ensures the database connection is valid
func (db *DB) Ping() error {
	err := db.ssn.Ping()
	if err != nil {
		return errors.Wrap(err, "could not contact the database")
	}
	//see if theres any permissions problems
	_, err = db.ssn.DatabaseNames()
	if err != nil {
		return errors.Wrap(err, "could not list the databases in the database")
	}
	return nil
}

//Close closing the underlying connection to MongoDB
func (db *DB) Close() {
	db.ssn.Close()
}

func dialTLS(mongoConfiguration config.MongoDB) (*mgo.Session, error) {
	tlsConf := tls.Config{
		InsecureSkipVerify: !mongoConfiguration.GetTLS().ShouldVerifyCertificate(),
	}
	caFilePath := mongoConfiguration.GetTLS().GetCAFile()
	if len(caFilePath) > 0 {
		pem, err := ioutil.ReadFile(caFilePath)
		err = errors.WithStack(err)
		if err != nil {
			return nil, errors.Wrap(err, "could not read CA file")
		}

		tlsConf.RootCAs = x509.NewCertPool()
		tlsConf.RootCAs.AppendCertsFromPEM(pem)
	}
	authMechanism, err := mongoConfiguration.GetAuthMechanism()
	if err != nil {
		return nil, errors.Wrap(err, "could not parse auth mechanism for MongoDB")
	}
	ssn, err := mgosec.Dial(mongoConfiguration.GetConnectionString(), authMechanism, &tlsConf)
	errors.WithStack(err)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to MongoDB")
	}
	return ssn, err
}

func dialInsecure(mongoConfiguration config.MongoDB) (*mgo.Session, error) {
	authMechanism, err := mongoConfiguration.GetAuthMechanism()
	if err != nil {
		return nil, errors.Wrap(err, "could not parse auth mechanism for MongoDB")
	}
	ssn, err := mgosec.DialInsecure(mongoConfiguration.GetConnectionString(), authMechanism)
	err = errors.WithStack(err)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to MongoDB")
	}
	return ssn, err
}
