package yaml

import (
	"testing"

	"github.com/activecm/mgosec"
	"github.com/stretchr/testify/require"
)

func TestNewYAMLConfig(t *testing.T) {
	testData := `MongoDB:
  ConnectionString: localhost:27018
  AuthenticationMechanism: SCRAM-SHA-1
  TLS:
    Enable: true
    VerifyCertificate: true
    CAFile: /etc/mycert
  Database: logstash
  Collection: ipfix
RITA:
  DBRoot: Collector1
  MetaDB: MetaDatabase
IPFIX:
  LocalNetworks:
    - 192.168.0.0/16
    - 172.16.0.0/12
    - 10.0.0.0/8
    - 195.154.15.32`

	testConfig, err := NewYAMLConfig([]byte(testData))
	require.Nil(t, err)

	require.Equal(t, "localhost:27018", testConfig.GetMongoDBConfig().GetConnectionString())

	mechanism, err := testConfig.GetMongoDBConfig().GetAuthMechanism()
	require.Nil(t, err)
	require.Equal(t, mgosec.ScramSha1, mechanism)

	require.True(t, testConfig.GetMongoDBConfig().GetTLS().IsEnabled())

	require.True(t, testConfig.GetMongoDBConfig().GetTLS().ShouldVerifyCertificate())

	require.Equal(t, "/etc/mycert", testConfig.GetMongoDBConfig().GetTLS().GetCAFile())

	require.Equal(t, "logstash", testConfig.GetMongoDBConfig().GetDatabase())

	require.Equal(t, "ipfix", testConfig.GetMongoDBConfig().GetCollection())

	require.Equal(t, "Collector1", testConfig.GetRITAConfig().GetDBRoot())

	require.Equal(t, "MetaDatabase", testConfig.GetRITAConfig().GetMetaDB())

	networks, errors := testConfig.GetIPFIXConfig().GetLocalNetworks()
	require.Len(t, errors, 1)
	require.Len(t, networks, 3)
	netStrings := make([]string, 0, len(networks))
	for i := range networks {
		netStrings = append(netStrings, networks[i].String())
	}
	require.ElementsMatch(t, []string{
		"192.168.0.0/16",
		"172.16.0.0/12",
		"10.0.0.0/8",
	}, netStrings)

}

func TestSaveYAMLConfig(t *testing.T) {
	testData := yamlConfig{
		MongoDB: mongoDB{
			ConnectionString: "localhost:27018",
			AuthMechanism:    "SCRAM-SHA-1",
			TLS: tls{
				Enabled:           true,
				VerifyCertificate: true,
				CAFile:            "/etc/mycert",
			},
			Database:   "logstash",
			Collection: "ipfix",
		},
		RITA: rita{
			DBRoot: "Collector1",
			MetaDB: "MetaDatabase",
		},
		IPFIX: ipfix{
			LocalNets: []string{"192.168.0.0/16"},
		},
	}
	serialData, err := testData.SaveConfig()
	require.Nil(t, err)
	newData := yamlConfig{}
	err = newData.LoadConfig(serialData)
	require.Nil(t, err)
	require.Equal(t, testData, newData)
}
