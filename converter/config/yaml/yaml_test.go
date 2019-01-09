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

Filtering:
    # These are filters that affect which flows are processed and which
    # are dropped.
    # A good reference for networks you may wish to consider is RFC 5735.
    # https://tools.ietf.org/html/rfc5735#section-4

    # Example: AlwaysInclude: ["192.168.1.2/32"]
    # This functionality overrides the NeverInclude and InternalSubnets
    # section, making sure that any connection records containing addresses from
    # this range are kept and not filtered.
    AlwaysInclude: []

    # Example: NeverInclude: ["255.255.255.255/32"]
    # This functions as a whitelisting setting, and connections involving
    # ranges entered into this section are filtered out.
    NeverInclude:
     - 0.0.0.0/32          # "This" Host           RFC 1122, Section 3.2.1.3
     - 127.0.0.0/8         # Loopback              RFC 1122, Section 3.2.1.3
     - 169.254.0.0/16      # Link Local            RFC 3927
     - 224.0.0.0/4         # Multicast             RFC 3171
     - 255.255.255.255/32  # Limited Broadcast     RFC 919, Section 7
     - ::1/128             # Loopback              RFC 4291, Section 2.5.3
     - fe80::/10           # Link local            RFC 4291, Section 2.5.6
     - ff00::/8            # Multicast             RFC 4291, Section 2.7

    # Example: InternalSubnets: ["10.0.0.0/8","172.16.0.0/12","192.168.0.0/16"]
    # This allows a user to identify their internal network, which will result
    # in any internal to internal and external to external connections being
    # filtered out at import time. Reasonable defaults are provided below
    # but need to be manually verified against each installation.
    InternalSubnets:
      - 10.0.0.0/8          # Private-Use Networks  RFC 1918
      - 172.16.0.0/12       # Private-Use Networks  RFC 1918
      - 192.168.0.0/16      # Private-Use Networks  RFC 1918
      - 195.154.15.32`

	testConfig, err := NewYAMLConfig([]byte(testData))
	require.Nil(t, err)

	logstashConf := testConfig.GetInputConfig().GetLogstashMongoDBConfig()
	testLogstashConfig(t, logstashConf)

	ritaConf := testConfig.GetOutputConfig().GetRITAConfig()
	testRITAConfig(t, ritaConf)

	filteringConf := testConfig.GetFilteringConfig()
	testFilteringConfig(t, filteringConf)
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

func testFilteringConfig(t *testing.T, filteringConf config.Filtering) {
	t.Run("Filtering Config", func(t *testing.T) {
		internalNets, errors := filteringConf.GetInternalSubnets()
		require.Len(t, internalNets, 3)
		require.Len(t, errors, 1)
		internalNetStrings := make([]string, 0, len(internalNets))
		for i := range internalNets {
			internalNetStrings = append(internalNetStrings, internalNets[i].String())
		}
		require.ElementsMatch(t, []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		}, internalNetStrings)

		alwaysIncludeNets, errors2 := filteringConf.GetAlwaysIncludeSubnets()
		require.Len(t, alwaysIncludeNets, 0)
		require.Len(t, errors2, 0)

		neverIncludeNets, errors3 := filteringConf.GetNeverIncludeSubnets()
		require.Len(t, neverIncludeNets, 8)
		require.Len(t, errors3, 0)
	})
}
