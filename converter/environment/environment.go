package environment

import (
	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/config/yaml"
	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/ipfix-rita/converter/logging"
)

//Environment is used to embed the methods provided by
//the logger, config manager, etc. into a given struct
//This alleviates passing around a method context/ resource bundle.
type Environment struct {
	config.Config
	logging.Logger
	DB database.DB
}

//NewDefaultEnvironment creates a new default environment
//reading the configuration from the standard yaml file,
//creating the pretty print logger, and connecting
//the database specified in the yaml configuration
func NewDefaultEnvironment() (Environment, error) {
	envOut := Environment{
		Logger: logging.NewLogrusLogger(),
	}
	configBuff, err := yaml.ReadConfigFile()
	if err != nil {
		return envOut, err
	}
	envOut.Config, err = yaml.NewYAMLConfig(configBuff)
	if err != nil {
		return envOut, err
	}
	envOut.DB, err = database.NewDB(envOut.GetMongoDBConfig(), envOut.GetRITAConfig())
	if err != nil {
		return envOut, err
	}
	return envOut, nil
}
