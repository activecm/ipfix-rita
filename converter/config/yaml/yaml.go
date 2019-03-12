package yaml

import (
	"bytes"
	"io"
	"os"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/pkg/errors"
	yaml2 "gopkg.in/yaml.v2"
)

//ConfigPath declares the location of the IPFIX-RITA yaml configuration file
const ConfigPath = "/etc/ipfix-rita/converter/converter.yaml"

//ReadConfigFile opens the ConfigPath for reading and returns the
//contents of the file or an error
func ReadConfigFile() ([]byte, error) {
	f, err := os.Open(ConfigPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not open configuration file")
	}
	defer f.Close()
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, f)
	if err != nil {
		return nil, errors.Wrap(err, "could not read configuration file")
	}
	return buf.Bytes(), nil
}

//yamlConfig contains the applications settings
//as represented by a YAML string. Implements config.Config
type yamlConfig struct {
	Input     input     `yaml:"Input"`
	Output    output    `yaml:"Output"`
	Filtering filtering `yaml:"Filtering"`
}

func (y *yamlConfig) GetInputConfig() config.Input {
	return &y.Input
}

func (y *yamlConfig) GetOutputConfig() config.Output {
	return &y.Output
}

func (y *yamlConfig) GetFilteringConfig() config.Filtering {
	return &y.Filtering
}

//NewYAMLConfig creates a new yamlConfig from
//a yaml string
func NewYAMLConfig(data []byte) (config.Config, error) {
	y := &yamlConfig{}
	err := y.LoadConfig(data)
	return y, errors.Wrap(err, "could not parse configuration")
}

func (y *yamlConfig) LoadConfig(data []byte) error {
	err := yaml2.Unmarshal(data, y)
	return errors.Wrapf(
		err, "could not unmarshal configuration\n"+
			"configuration: %s\n"+
			"destination type: %T", data, *y,
	)
}

func (y *yamlConfig) SaveConfig() ([]byte, error) {
	outBytes, err := yaml2.Marshal(y)
	return outBytes, errors.Wrapf(err, "could not marshal configuration:\n%+v", y)
}
