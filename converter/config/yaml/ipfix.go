package yaml

import "net"
import "github.com/pkg/errors"

//ipfix implements config.IPFIX
type ipfix struct {
	LocalNets []string `yaml:"LocalNetworks"`
}

func (i *ipfix) GetLocalNetworks() ([]net.IPNet, []error) {
	var errorList []error
	var nets []net.IPNet
	for j := range i.LocalNets {
		_, net, err := net.ParseCIDR(i.LocalNets[j])
		if err != nil {
			err = errors.WithStack(err)
			errorList = append(errorList, err)
		} else {
			nets = append(nets, *net)
		}
	}
	return nets, errorList
}
