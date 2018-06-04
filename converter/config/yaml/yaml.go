package yaml

import (
	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/pkg/errors"
	yaml2 "gopkg.in/yaml.v2"
)

//yamlConfig contains the applications settings
//as represented by a YAML string
type yamlConfig struct {
	MongoDB mongoDB `yaml:"MongoDB"`
	RITA    rita    `yaml:"RITA"`
	IPFIX   ipfix   `yaml:"IPFIX"`
}

func (y *yamlConfig) GetMongoDBConfig() config.MongoDB {
	return &y.MongoDB
}

func (y *yamlConfig) GetRITAConfig() config.RITA {
	return &y.RITA
}

func (y *yamlConfig) GetIPFIXConfig() config.IPFIX {
	return &y.IPFIX
}

//NewYAMLConfig creates a new yamlConfig from
//a yaml string
func NewYAMLConfig(data []byte) (config.Config, error) {
	y := &yamlConfig{}
	err := y.LoadConfig(data)
	return y, err
}

func (y *yamlConfig) LoadConfig(data []byte) error {
	err := yaml2.Unmarshal(data, y)
	err = errors.WithStack(err)
	return err
}

func (y *yamlConfig) SaveConfig() ([]byte, error) {
	outBytes, err := yaml2.Marshal(y)
	err = errors.WithStack(err)
	return outBytes, err
}
