package yaml

import (
	"testing"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/mgosec"
	"github.com/stretchr/testify/require"
)

func TestNewYAMLConfig(t *testing.T) {
	testData := `Input:
  Logstash-MongoDB:
    MongoDB-Connection:
      # See https://docs.mongodb.com/manual/reference/connection-string/
      ConnectionString: mongodb://mongodb:27017
      # Accepted Values: null, "SCRAM-SHA-1", "MONGODB-CR"
      AuthenticationMechanism: null
      TLS:
        Enable: false
        VerifyCertificate: false
        CAFile: null

    # The database and collection holding records produced by the collector
    Database: IPFIX
    Collection: in

Output:
  RITA-MongoDB:
    MongoDB-Connection:
      # See https://docs.mongodb.com/manual/reference/connection-string/
      ConnectionString: mongodb://mongodb:27018
      # Accepted Values: null, "SCRAM-SHA-1", "MONGODB-CR"
      AuthenticationMechanism: SCRAM-SHA-1
      TLS:
        Enable: true
        VerifyCertificate: true
        CAFile: /path/to/CAFile

    # The resulting RITA databases will be displayed as "DBRoot-YYYY-MM-DD"
    DBRoot: IPFIX-OUT

    # This database holds information about RITA managed databases.
    MetaDB: MetaDatabase

IPFIX:
  # CIDR ranges of networks to mark local
  LocalNetworks:
    - 192.168.0.0/16
    - 172.16.0.0/12
    - 10.0.0.0/8
    - 195.154.15.32`

	testConfig, err := NewYAMLConfig([]byte(testData))
	require.Nil(t, err)

	logstashConf := testConfig.GetInputConfig().GetLogstashMongoDBConfig()
	testLogstashConfig(t, logstashConf)

	ritaConf := testConfig.GetOutputConfig().GetRITAConfig()
	testRITAConfig(t, ritaConf)

	ipfixConf := testConfig.GetIPFIXConfig()
	testIPFIXConfig(t, ipfixConf)
}

func testLogstashConfig(t *testing.T, logstashConf config.LogstashMongoDB) {
	t.Run("Logstash-MongoDB Config", func(t *testing.T) {
		require.Equal(t, "mongodb://mongodb:27017", logstashConf.GetConnectionConfig().GetConnectionString())
		mechanism, err := logstashConf.GetConnectionConfig().GetAuthMechanism()
		require.Nil(t, err)
		require.Equal(t, mgosec.None, mechanism)
		require.False(t, logstashConf.GetConnectionConfig().GetTLS().IsEnabled())
		require.False(t, logstashConf.GetConnectionConfig().GetTLS().ShouldVerifyCertificate())
		require.Equal(t, "", logstashConf.GetConnectionConfig().GetTLS().GetCAFile())
		require.Equal(t, "IPFIX", logstashConf.GetDatabase())
		require.Equal(t, "in", logstashConf.GetCollection())
	})
}

func testRITAConfig(t *testing.T, ritaConf config.RITA) {
	t.Run("RITA-MongoDB Config", func(t *testing.T) {
		require.Equal(t, "mongodb://mongodb:27018", ritaConf.GetConnectionConfig().GetConnectionString())
		mechanism, err := ritaConf.GetConnectionConfig().GetAuthMechanism()
		require.Nil(t, err)
		require.Equal(t, mgosec.ScramSha1, mechanism)
		require.True(t, ritaConf.GetConnectionConfig().GetTLS().IsEnabled())
		require.True(t, ritaConf.GetConnectionConfig().GetTLS().ShouldVerifyCertificate())
		require.Equal(t, "/path/to/CAFile", ritaConf.GetConnectionConfig().GetTLS().GetCAFile())
		require.Equal(t, "IPFIX-OUT", ritaConf.GetDBRoot())
		require.Equal(t, "MetaDatabase", ritaConf.GetMetaDB())
	})
}

func testIPFIXConfig(t *testing.T, ipfixConf config.IPFIX) {
	t.Run("IPFIX Config", func(t *testing.T) {
		networks, errors := ipfixConf.GetLocalNetworks()
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
	})
}
