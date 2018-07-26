package integrationtest

import (
	"net"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/mgosec"
)

//newConfig returns a standard testing configuration
//linked to a given instance of MongoDB via the mongoDBURI.
//MongoDB must be run without encryption/ authentication.
func newConfig(mongoDBURI string) config.Config {
	return testConfig{
		mongoDB: mongoDBConfig{
			connectionString: mongoDBURI,
		},
	}
}

//testConfig implements Config
type testConfig struct {
	mongoDB mongoDBConfig
	ipfix   ipfixConfig
	rita    ritaConfig
}

func (t testConfig) GetMongoDBConfig() config.MongoDB { return t.mongoDB }
func (t testConfig) GetIPFIXConfig() config.IPFIX     { return t.ipfix }
func (t testConfig) GetRITAConfig() config.RITA       { return t.rita }

//mongoDBConfig implements config.MongoDB
type mongoDBConfig struct {
	connectionString string
	tls              testingTLSConfig
}

func (m mongoDBConfig) GetConnectionString() string { return m.connectionString }
func (m mongoDBConfig) GetAuthMechanism() (mgosec.AuthMechanism, error) {
	return mgosec.None, nil
}
func (m mongoDBConfig) GetTLS() config.TLS { return m.tls }

func (m mongoDBConfig) GetDatabase() string   { return "IPFIX" }
func (m mongoDBConfig) GetCollection() string { return "in" }

//testingTLSConfig implements config.TLS
type testingTLSConfig struct{}

func (t testingTLSConfig) IsEnabled() bool               { return false }
func (t testingTLSConfig) ShouldVerifyCertificate() bool { return false }
func (t testingTLSConfig) GetCAFile() string             { return "" }

//ipfixConfig implements config.IPFIX
type ipfixConfig struct{}

func (t ipfixConfig) GetLocalNetworks() ([]net.IPNet, []error) {
	return []net.IPNet{}, []error{}
}

//TestingRITAConfig implements config.RITA
type ritaConfig struct{}

func (r ritaConfig) GetDBRoot() string { return "RITA" }
func (r ritaConfig) GetMetaDB() string { return "MetaDatabase" }
