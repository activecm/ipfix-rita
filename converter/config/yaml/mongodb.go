package yaml

import (
	"time"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/mgosec"
	"github.com/pkg/errors"
)

//mongoDB implements config.MongoDB
type mongoDB struct {
	ConnectionString string        `yaml:"ConnectionString"`
	AuthMechanism    string        `yaml:"AuthenticationMechanism"`
	SocketTimeout    time.Duration `yaml:"SocketTimeout"`
	TLS              tls           `yaml:"TLS"`
}

func (m *mongoDB) GetConnectionString() string {
	return m.ConnectionString
}

func (m *mongoDB) GetAuthMechanism() (mgosec.AuthMechanism, error) {
	mechanism, err := mgosec.ParseAuthMechanism(m.AuthMechanism)
	err = errors.WithStack(err)
	return mechanism, err
}

func (m *mongoDB) GetSocketTimeout() time.Duration {
	return m.SocketTimeout * time.Hour
}

func (m *mongoDB) GetTLS() config.TLS {
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
