package database

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/mgosec"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
)

const sessionsCollection = "sessions"

//DB extends mgo.Session with application specific functionality
type DB struct {
	ssn      *mgo.Session
	input    *mgo.Collection
	sessions *mgo.Collection
}

//NewDB creates a mongodb session based on the mongo config
func NewDB(conf config.MongoDB) (DB, error) {
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
