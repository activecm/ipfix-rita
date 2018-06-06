package integrationtesting

import (
	"net"
	"time"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/mgosec"
)

//NewTestingConfig returns a standard testing configuration
//linked to a given instance of MongoDB via the mongoDBURI.
//MongoDB must be run without encryption/ authentication.
func NewTestingConfig(mongoDBURI string) config.Config {
	return testingConfig{
		mongoDB: testingMongoDBConfig{
			connectionString: mongoDBURI,
		},
	}
}

//testingConfig implements Config
type testingConfig struct {
	mongoDB testingMongoDBConfig
	ipfix   testingIPFIXConfig
	rita    testingRITAConfig
}

func (t testingConfig) GetMongoDBConfig() config.MongoDB { return t.mongoDB }
func (t testingConfig) GetIPFIXConfig() config.IPFIX     { return t.ipfix }
func (t testingConfig) GetRITAConfig() config.RITA       { return t.rita }

//testingMongoDBConfig implements config.MongoDB
type testingMongoDBConfig struct {
	connectionString string
	tls              testingTLSConfig
}

func (m testingMongoDBConfig) GetConnectionString() string { return m.connectionString }
func (m testingMongoDBConfig) GetAuthMechanism() (mgosec.AuthMechanism, error) {
	return mgosec.None, nil
}
func (m testingMongoDBConfig) GetSocketTimeout() time.Duration { return 5 * time.Minute }
func (m testingMongoDBConfig) GetTLS() config.TLS              { return m.tls }

//testingTLSConfig implements config.TLS
type testingTLSConfig struct{}

func (t testingTLSConfig) IsEnabled() bool               { return false }
func (t testingTLSConfig) ShouldVerifyCertificate() bool { return false }
func (t testingTLSConfig) GetCAFile() string             { return "" }

//testingIPFIXConfig implements config.IPFIX
type testingIPFIXConfig struct{}

func (t testingIPFIXConfig) GetDatabase() string   { return "IPFIX" }
func (t testingIPFIXConfig) GetCollection() string { return "in" }
func (t testingIPFIXConfig) GetLocalNetworks() ([]net.IPNet, []error) {
	return []net.IPNet{}, []error{}
}

//TestingRITAConfig implements config.RITA
type testingRITAConfig struct{}

func (r testingRITAConfig) GetDBRoot() string { return "RITA" }
