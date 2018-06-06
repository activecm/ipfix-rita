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

//DB extends mgo.Session with application specific functionality
type DB struct {
	*mgo.Session
}

//NewDB creates a mongodb session based on the mongo config
func NewDB(conf config.MongoDB) (DB, error) {
	db := DB{}
	var err error
	if conf.GetTLS().IsEnabled() {
		db.Session, err = dialTLS(conf)
	} else {
		db.Session, err = dialInsecure(conf)
	}
	if err != nil {
		return db, err
	}
	db.SetSocketTimeout(conf.GetSocketTimeout())
	db.SetCursorTimeout(conf.GetSocketTimeout())
	db.SetSyncTimeout(conf.GetSocketTimeout())
	return db, nil
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
