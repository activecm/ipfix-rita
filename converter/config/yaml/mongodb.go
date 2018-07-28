package yaml

import (
	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/mgosec"
	"github.com/pkg/errors"
)

//mongoDBConnection implements config.MongoDBConnection
type mongoDBConnection struct {
	ConnectionString string `yaml:"ConnectionString"`
	AuthMechanism    string `yaml:"AuthenticationMechanism"`
	TLS              tls    `yaml:"TLS"`
}

func (m *mongoDBConnection) GetConnectionString() string {
	return m.ConnectionString
}

func (m *mongoDBConnection) GetAuthMechanism() (mgosec.AuthMechanism, error) {
	mechanism, err := mgosec.ParseAuthMechanism(m.AuthMechanism)
	return mechanism, errors.Wrapf(err, "could not parse MongoDB AuthMechanism: %s", m.AuthMechanism)
}

func (m *mongoDBConnection) GetTLS() config.TLS {
	return &m.TLS
}

//tls implements config.TLS
type tls struct {
	Enabled           bool   `yaml:"Enable"`
	VerifyCertificate bool   `yaml:"VerifyCertificate"`
	CAFile            string `yaml:"CAFile"`
}

func (t *tls) IsEnabled() bool {
	return t.Enabled
}

func (t *tls) ShouldVerifyCertificate() bool {
	return t.VerifyCertificate
}

func (t *tls) GetCAFile() string {
	return t.CAFile
}
