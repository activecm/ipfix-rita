package mongodb

import (
	"time"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/database"
	mgo "github.com/globalsign/mgo"
	"github.com/pkg/errors"
)

//LogstashMongoInputDB wraps a *mgo.Database connected
//to a MongoDB database containing Logstash decoded IPFIX/ Netflow records
type LogstashMongoInputDB struct {
	db              *mgo.Database
	inputCollection string
}

//NewLogstashMongoInputDB instantiates a new LogstashMongoInputDB
//from the LogstashMongoDB configuration
func NewLogstashMongoInputDB(logstashMongo config.LogstashMongoDB) (LogstashMongoInputDB, error) {
	db := LogstashMongoInputDB{}
	ssn, err := database.Dial(logstashMongo.GetConnectionConfig())
	if err != nil {
		return db, err
	}
	ssn.SetSocketTimeout(1 * time.Hour)
	ssn.SetSyncTimeout(1 * time.Hour)
	ssn.SetCursorTimeout(0)

	db.db = ssn.DB(logstashMongo.GetDatabase())
	db.inputCollection = logstashMongo.GetCollection()
	return db, nil
}

//NewHelperCollection returns a new socket connected
//to a MongoDB collection in the input database with the given name collName.
func (i LogstashMongoInputDB) NewHelperCollection(collName string) *mgo.Collection {
	return i.db.C(collName).With(i.db.Session.Copy())
}

//NewInputConnection returns a new socket connected to the input
//collection
func (i LogstashMongoInputDB) NewInputConnection() *mgo.Collection {
	return i.db.C(i.inputCollection).With(i.db.Session.Copy())
}

//Ping ensures the database connection is valid
func (i LogstashMongoInputDB) Ping() error {
	err := i.db.Session.Ping()
	if err != nil {
		return errors.Wrap(err, "could not contact the database")
	}
	//see if theres any permissions problems
	_, err = i.db.Session.DatabaseNames()
	if err != nil {
		return errors.Wrap(err, "could not list the databases in the database")
	}
	return nil
}

//Close closing the underlying connection to MongoDB
func (i LogstashMongoInputDB) Close() {
	i.db.Session.Close()
}
