package config

import (
	"net"
	"time"

	"github.com/activecm/mgosec"
)

//ADDING A CONFIGURATION OPTION:
//A new configuration option should belong to a parent scope.
//If the scope does not exist, create an interface here
//exposing each of the configuration options for that scope
//as method calls. Then, expose that interface via
//the Config interface. You may also need to edit the struct
//implementing Config if you created a new scope.
//
//If the scope already exists, simply add a method exposing
//the option to that scope's interface.
//
//Next, for each Config implementation (yamlConfig and testConfig),
//find the object backing the scope interface which has been modified,
//add the field to struct definition and add a getter.
//
//Finally, make sure to edit the tests for each format.

//Config holds IPFIX-RITA configuration details
type Config interface {
	GetMongoDBConfig() MongoDB
	GetRITAConfig() RITA
	GetIPFIXConfig() IPFIX
}

//Serializable represents application configuration data
//which can be serialized (to YAML for example)
type Serializable interface {
	Config
	LoadConfig(data []byte) error
	SaveConfig() ([]byte, error)
}

//MongoDB provides information for contacting MongoDB
type MongoDB interface {
	GetConnectionString() string
	GetAuthMechanism() (mgosec.AuthMechanism, error)
	GetSocketTimeout() time.Duration
	GetTLS() TLS
}

//TLS provides information for contacting MongoDB over TLS
type TLS interface {
	IsEnabled() bool
	ShouldVerifyCertificate() bool
	GetCAFile() string
}

//RITA provides information for coordinating with RITA
type RITA interface {
	GetDBRoot() string
}

//IPFIX provides information for accessing IPFIX data
//and information regarding the individual records
type IPFIX interface {
	GetDatabase() string
	GetCollection() string
	GetLocalNetworks() ([]net.IPNet, []error)
}
