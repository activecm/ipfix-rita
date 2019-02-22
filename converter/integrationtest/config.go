package integrationtest

import (
	"net"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/mgosec"
)

//TestConfig implements config.Config
type TestConfig struct {
	input     InputConfig
	output    OutputConfig
	filtering FilteringConfig
}

func (t *TestConfig) GetInputConfig() config.Input         { return &t.input }
func (t *TestConfig) GetOutputConfig() config.Output       { return &t.output }
func (t *TestConfig) GetFilteringConfig() config.Filtering { return &t.filtering }

//InputConfig implements config.Input
type InputConfig struct {
	logstashMongo LogstashMongoConfig
}

func (t *InputConfig) GetLogstashMongoDBConfig() config.LogstashMongoDB { return &t.logstashMongo }

//LogstashMongoConfig implements config.LogstashMongoDB
type LogstashMongoConfig struct {
	mongoDB MongoDBConfig
}

func (t *LogstashMongoConfig) GetConnectionConfig() config.MongoDBConnection { return &t.mongoDB }
func (t *LogstashMongoConfig) GetDatabase() string                           { return "IPFIX" }
func (t *LogstashMongoConfig) GetCollection() string                         { return "in" }

//MongoDBConfig implements config.MongoDB
type MongoDBConfig struct {
	connectionString string
	tls              TLSConfig
}

func (m *MongoDBConfig) GetConnectionString() string { return m.connectionString }

//SetConnectionString provides a private setter
//so the connection string can be updated dynamically
func (m *MongoDBConfig) SetConnectionString(connectionString string) {
	m.connectionString = connectionString
}

func (m *MongoDBConfig) GetAuthMechanism() (mgosec.AuthMechanism, error) {
	return mgosec.None, nil
}
func (m *MongoDBConfig) GetTLS() config.TLS { return &m.tls }

//TLSConfig implements config.TLS
type TLSConfig struct{}

func (t *TLSConfig) IsEnabled() bool               { return false }
func (t *TLSConfig) ShouldVerifyCertificate() bool { return false }
func (t *TLSConfig) GetCAFile() string             { return "" }

//OutputConfig implements config.Output
type OutputConfig struct {
	rita RitaConfig
}

func (t *OutputConfig) GetRITAConfig() config.RITA { return &t.rita }

//RitaConfig implements config.RITA
type RitaConfig struct {
	mongoDB MongoDBConfig
	strobe  StrobeConfig
}

func (r *RitaConfig) GetConnectionConfig() config.MongoDBConnection { return &r.mongoDB }

func (r *RitaConfig) GetDBRoot() string        { return "RITA" }
func (r *RitaConfig) GetMetaDB() string        { return "MetaDatabase" }
func (r *RitaConfig) GetStrobe() config.Strobe { return &r.strobe }

type StrobeConfig struct{}

func (s *StrobeConfig) GetConnectionLimit() int {
	//Lowered from the usual 250000 so tests involving this limit don't take forever.
	return 100
}

//FilteringConfig implements config.Filtering
type FilteringConfig struct{}

func (f *FilteringConfig) GetAlwaysIncludeSubnets() ([]net.IPNet, []error) {
	return []net.IPNet{}, []error{}
}

func (f *FilteringConfig) GetNeverIncludeSubnets() ([]net.IPNet, []error) {
	return []net.IPNet{}, []error{}
}

func (f *FilteringConfig) GetInternalSubnets() ([]net.IPNet, []error) {
	return []net.IPNet{}, []error{}
}
