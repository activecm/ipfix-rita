package database

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/mgosec"
	mgo "github.com/globalsign/mgo"
	"github.com/pkg/errors"
)

//Dial connects to MongoDB as specified in the mongoConfiguration
//and returns a valid *mgo.Session or an error
func Dial(mongoConfiguration config.MongoDBConnection) (*mgo.Session, error) {
	var err error
	var ssn *mgo.Session
	if mongoConfiguration.GetTLS().IsEnabled() {
		ssn, err = dialTLS(mongoConfiguration)
		err = errors.Wrap(err, "could not connect to MongoDB over TLS")
	} else {
		ssn, err = dialInsecure(mongoConfiguration)
		err = errors.Wrap(err, "could not connect to MongoDB (no TLS)")
	}

	//HACK: retry tyhe connection a few times
	//Proper way to resolve this is to alter mgosec to accept a custom timeout
	//The default timeout is 5 seconds. Bump it up to 30 by repeating 6 times.
	for i := 0; err != nil && errors.Cause(err).Error() == "no reachable servers" && i < 6; i++ {
		if mongoConfiguration.GetTLS().IsEnabled() {
			ssn, err = dialTLS(mongoConfiguration)
			err = errors.Wrap(err, "could not connect to MongoDB over TLS")
		} else {
			ssn, err = dialInsecure(mongoConfiguration)
			err = errors.Wrap(err, "could not connect to MongoDB (no TLS)")
		}
	}
	return ssn, err
}

func dialTLS(mongoConfiguration config.MongoDBConnection) (*mgo.Session, error) {
	tlsConf := tls.Config{
		InsecureSkipVerify: !mongoConfiguration.GetTLS().ShouldVerifyCertificate(),
	}
	caFilePath := mongoConfiguration.GetTLS().GetCAFile()
	if len(caFilePath) > 0 {
		pem, err := ioutil.ReadFile(caFilePath)
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
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to MongoDB")
	}
	return ssn, err
}

func dialInsecure(mongoConfiguration config.MongoDBConnection) (*mgo.Session, error) {
	authMechanism, err := mongoConfiguration.GetAuthMechanism()
	if err != nil {
		return nil, errors.Wrap(err, "could not parse auth mechanism for MongoDB")
	}
	ssn, err := mgosec.DialInsecure(mongoConfiguration.GetConnectionString(), authMechanism)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to MongoDB")
	}
	return ssn, err
}
