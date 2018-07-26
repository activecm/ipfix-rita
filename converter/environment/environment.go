package environment

import (
	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/config/yaml"
	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/pkg/errors"
)

//Environment is used to embed the methods provided by
//the logger, config manager, etc. into a given struct
//The Environment is used like a resource bundle.
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
		return envOut, errors.Wrap(err, "could not read configuration file")
	}
	envOut.Config, err = yaml.NewYAMLConfig(configBuff)
	if err != nil {
		return envOut, errors.Wrap(err, "could not parse configuration")
	}
	envOut.DB, err = database.NewDB(envOut.GetMongoDBConfig(), envOut.GetRITAConfig())
	//HACK: retry tyhe connection a few times
	//Proper way to resolve this is to alter mgosec to accept a custom timeout
	//The default timeout is 5 seconds. Bump it up to 30 by repeating 6 times.
	for i := 0; err != nil && errors.Cause(err).Error() == "no reachable servers" && i < 6; i++ {
		envOut.DB, err = database.NewDB(envOut.GetMongoDBConfig(), envOut.GetRITAConfig())
		envOut.Logger.Warn("could not reach MongoDB server. retrying...", nil)
	}
	if err != nil {
		return envOut, errors.Wrap(err, "could not connect to database specified in configuration")
	}
	return envOut, nil
}
