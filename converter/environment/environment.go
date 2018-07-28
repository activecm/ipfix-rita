package environment

import (
	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/config/yaml"
	"github.com/activecm/ipfix-rita/converter/logging"
	"github.com/pkg/errors"
)

//Environment is used to embed the methods provided by
//the logger, config manager, etc. into a given struct
//The Environment is used like a resource bundle.
type Environment struct {
	config.Config
	logging.Logger
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
	return envOut, nil
}
