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

const sessionsCollection = "sessions"
const metaDBDatabasesCollection = "databases"
const ritaConnInputCollection = "conn"

//DB extends mgo.Session with application specific functionality
type DB struct {
	ssn             *mgo.Session
	input           *mgo.Collection
	sessions        *mgo.Collection
	metaDBDatabases *mgo.Collection
	ritaConf        config.RITA
}

//NewDB creates a mongodb session based on the mongo config
func NewDB(conf config.MongoDB, ritaConf config.RITA) (DB, error) {
	db := DB{}
	var err error
	if conf.GetTLS().IsEnabled() {
		db.ssn, err = dialTLS(conf)
	} else {
		db.ssn, err = dialInsecure(conf)
	}
	if err != nil {
		return db, err
	}
	db.ssn.SetSocketTimeout(conf.GetSocketTimeout())
	db.ssn.SetSyncTimeout(conf.GetSocketTimeout())
	db.ssn.SetCursorTimeout(0)

	db.input = db.ssn.DB(conf.GetDatabase()).C(conf.GetCollection())
	db.sessions = db.ssn.DB(conf.GetDatabase()).C(sessionsCollection)
	db.metaDBDatabases = db.ssn.DB(ritaConf.GetMetaDB()).C(metaDBDatabasesCollection)
	db.ritaConf = ritaConf

	err = db.sessions.EnsureIndex(mgo.Index{
		Key: []string{
			"IPAddressA", "transportPortA",
			"IPAddressB", "transportPortB",
			"protocolIdentifier", "exporter",
		},
		Unique: true,
		Name:   "AggregateQuery",
	})

	if err != nil {
		return db, err
	}

	err = db.sessions.EnsureIndex(mgo.Index{
		Key: []string{
			"exporter",
			"flowEndMillisecondsAB",
			"flowEndMillisecondsBA",
		},
		Name: "ExpirationQuery",
	})

	if err != nil {
		return db, err
	}

	err = db.metaDBDatabases.EnsureIndex(mgo.Index{
		Key: []string{
			"name",
		},
		Unique:   true,
		DropDups: true,
		Name:     "nameindex",
	})

	if err != nil {
		return db, err
	}

	return db, nil
}

//NewInputConnection returns a new socket connected to the input
//collection
func (db *DB) NewInputConnection() *mgo.Collection {
	ssn := db.ssn.Copy()
	return db.input.With(ssn)
}

//NewSessionsConnection returns a new socket connected to the
//session aggregate collection
func (db *DB) NewSessionsConnection() *mgo.Collection {
	ssn := db.ssn.Copy()
	return db.sessions.With(ssn)
}

//NewMetaDBDatabasesConnection returns a new socket connected to the
//MetaDB databases collection
func (db *DB) NewMetaDBDatabasesConnection() *mgo.Collection {
	ssn := db.ssn.Copy()
	return db.metaDBDatabases.With(ssn)
}

//NewOutputConnection returns a new socket connected to the
//RITA output collection with a given DB suffix
func (db *DB) NewOutputConnection(suffix string) (*mgo.Collection, error) {
	ssn := db.ssn.Copy()
	dbName := db.ritaConf.GetDBRoot()
	if suffix != "" {
		dbName = db.ritaConf.GetDBRoot() + "-" + suffix
	}
	connColl := ssn.DB(dbName).C(ritaConnInputCollection)
	tmpConn := parsetypes.Conn{}
	for _, index := range tmpConn.Indices() {
		err := connColl.EnsureIndex(mgo.Index{
			Key: []string{index},
		})

		if err != nil {
			ssn.Close()
			return nil, err
		}
	}
	return connColl, nil
}

//Ping ensures the database connection is valid
func (db *DB) Ping() error {
	err := db.ssn.Ping()
	if err != nil {
		return err
	}
	//see if theres any permissions problems
	_, err = db.ssn.DatabaseNames()
	if err != nil {
		return err
	}
	return nil
}

//Close closing the underlying connection to MongoDB
func (db *DB) Close() {
	db.ssn.Close()
}

func dialTLS(conf config.MongoDB) (*mgo.Session, error) {
	tlsConf := tls.Config{
		InsecureSkipVerify: !conf.GetTLS().ShouldVerifyCertificate(),
	}
	caFilePath := conf.GetTLS().GetCAFile()
	if len(caFilePath) > 0 {
		pem, err := ioutil.ReadFile(caFilePath)
		err = errors.WithStack(err)
		if err != nil {
			return nil, err
		}

		tlsConf.RootCAs = x509.NewCertPool()
		tlsConf.RootCAs.AppendCertsFromPEM(pem)
	}
	authMechanism, err := conf.GetAuthMechanism()
	if err != nil {
		return nil, err
	}
	ssn, err := mgosec.Dial(conf.GetConnectionString(), authMechanism, &tlsConf)
	errors.WithStack(err)
	if err != nil {
		return nil, err
	}
	return ssn, err
}

func dialInsecure(conf config.MongoDB) (*mgo.Session, error) {
	authMechanism, err := conf.GetAuthMechanism()
	if err != nil {
		return nil, err
	}
	ssn, err := mgosec.DialInsecure(conf.GetConnectionString(), authMechanism)
	err = errors.WithStack(err)
	if err != nil {
		return nil, err
	}
	return ssn, err
}
