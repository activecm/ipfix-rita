package config

import (
	"net"

	"github.com/activecm/mgosec"
)

//ADDING A CONFIGURATION OPTION:
//A new configuration option should belong to a parent scope.
//If the scope does not exist, create an interface here
//exposing each of the configuration options for that scope
//as method calls. Then, expose that interface via
//the Config interface. You will also need to edit the struct(s)
//implementing Config if you created a new scope. (explained below)
//
//If the scope already exists, simply add a method exposing
//the option to that scope's interface.
//
//Next, for each Config implementation (yamlConfig and testConfig),
//find the object backing the scope interface which has been modified,
//add the field to struct definition and add a getter.
//
//Finally, make sure to edit the tests for each format.

//Config holds IPFIX-RITA (converter) configuration details
type Config interface {
	GetInputConfig() Input
	GetFilteringConfig() Filtering
	GetOutputConfig() Output
}

//Serializable represents application configuration data
//which can be serialized (to YAML for example)
type Serializable interface {
	Config
	LoadConfig(data []byte) error
	SaveConfig() ([]byte, error)
}

//MongoDBConnection provides information for contacting MongoDB\
type MongoDBConnection interface {
	GetConnectionString() string
	GetAuthMechanism() (mgosec.AuthMechanism, error)
	GetTLS() TLS
}

//TLS provides information for contacting MongoDB over TLS
type TLS interface {
	IsEnabled() bool
	ShouldVerifyCertificate() bool
	GetCAFile() string
}

//Input contains configuration for ingesting IPFIX/ Netflow data
type Input interface {
	GetLogstashMongoDBConfig() LogstashMongoDB
}

//LogstashMongoDB contains configuration for ingesting Logstash
//decoded IPFIX/ Netflow records from MongoDB
type LogstashMongoDB interface {
	GetConnectionConfig() MongoDBConnection
	GetDatabase() string
	GetCollection() string
}

//Output contains configuration for writing out the
//stitched IPFIX/ Netflow records
type Output interface {
	GetRITAConfig() RITA
}

//RITA contains configuration for writing out the
//stitched IPFIX/ Netflow records RITA compatible MongoDB databases
type RITA interface {
	GetConnectionConfig() MongoDBConnection
	GetDBRoot() string
	GetMetaDB() string
}

//Filtering contains information on local subnets and other networks/hosts
//that should be filtered out of the result set
type Filtering interface {
	GetAlwaysIncludeSubnets() ([]net.IPNet, []error)
	GetNeverIncludeSubnets() ([]net.IPNet, []error)
	GetInternalSubnets() ([]net.IPNet, []error)
}
